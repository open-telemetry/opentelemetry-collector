# Avito Quota Processor

Получает квоты из внешнего источника и применяет их к потоку логов.

## Структура

### Quota Processor

Принимает логи и пытается получить для них правило квоты. Если правило найдено, то квота применяется к логу. Если правило не найдено, то используется дефолтная квота.

Когда определен TTL, вычисляется поля `observer.ttl_bucket_name`, `observer.ttl_timestamp_unix_s`, `observer.partition_name` и добавляются в атрибуты лога.

### Rules

Квоты задаются правилами. В сыром виде их довольно проблематично матчить на каждый лог, поэтому они преобразуются в два этапа:

1. Все правила разбираются на группы по трем ключам: `service`, `severity`, `system`. Это позволяет быстро отфильтровать правила, в которых нет этих ключей.
2. Внутри группы правила раскладываются в специальную структуру, которая позволяет быстро матчить их на каждый лог. Для этого делается обратный индекс по выражениям в правилах. Как только у правила матчатся все выражения, то оно считается подходящим. После чего правило проверяется на квоту. Если квота превышена, то правило считается не подходящим и продолжается поиск другого правила.

### Limiter

Для каждого правила создается отдельный лимитер. Лимитер хранит в себе информацию о том, сколько логов (или, например, стораджа) уже было использовано по этому правилу. Лимитер обновляется каждый раз, когда приходит новый лог.

### Syncer

Syncer отвечает за синхронизацию правил с внешним хранилищем. Ранее, использовался Redis. В Q4Y25 правила переехали в хранилище в сервисе paas-observabiliy, откуда мы их получаем через ручку [listLogsRules](https://paas.k.avito.ru/services/service-paas-observability?tab=brief&env=prod&briefTab=produce&briefChapter=rpc&briefAnchor=listLogsRules&region=ru).
Syncer периодически проверяет, изменились ли правила. Если изменились, то Syncer обновляет правила в памяти. И рассылает их всем клиентам. Для обновления используется оптимизация: все правила хранятся в atomic pointer, и при обновлении создается новый указатель на новые правила. Причем сами правила, если они не изменились, остаются те же. Это позволяет избежать лишних блокировок и копирований.

## Config

- `loglevel` - уровень логирования
- `rules_update_interval` - интервал обновления правил
- `default_ttl_days` - значение по умолчанию для TTL в днях
- `client_tls` - клиентский сертификат который позоволит подключаться к сервису paas-observability
  - `enabled`
  - `crt_path`
  - `key_path`

Ожидается, что правила будут возвращаться ручкой в формате массива правил

```json
[
  {
    "ruleID": "01b59250-5039-4790-8cbc-595d0580b480",
    "filter": [
      {
        "key": {
          "name": "service",
          "kind": "system"
        },
        "operator": "=",
        "value": "vertica-guard"
      }
    ],
    "quotas": [
      {
        "resourceMetricID": "logsPerSec",
        "value": 1000
      },
      {
        "resourceMetricID": "logsStorage",
        "value": 1000000
      }
    ],
    "ttl": {
      "id": "14d",
      "resourceId": "ch-opentracing",
      "name": "14d",
      "durationSeconds": 1296000,
      "isDefault": true
    }
  },
  {
  }
]
```

## Rules Internals

### Rules preprocessing

#### Rules Inverted Index

```go
type Rule struct {
    ID             string       `json:"id"`
    Name           string       `json:"name"`
    Expressions    []Expression `json:"expressions"`

    RatePerSecond     int `json:"rate_per_second"`
    StorageRatePerDay int `json:"storage_rate_per_day"`
    RetentionDays     int `json:"retention_days"`
}

type Expression struct {
    Key       string `json:"key"`
    Operation string `json:"operation"`
    Value     string `json:"value"`
}
```

Правила приходят в виде JSON, который парсится в структуру `Rule`. После чего мы подготавливаем правила для дальнейшей работы.

Для этого мы обрабатываем массив []*Rule и достаем из него упордоченный массив выражений и индекс, который указывает на правила, которые содержат это выражение.

Пример:

```txt
Rule {
    ID: "ololo",
    Name: "test",
    Expressions: [
        {
            Key: "key",
            Operation: "eq",
            Value: "value"
        }
    ],
    RatePerSecond: 10,
    StorageRatePerDay: 100,
    RetentionDays: 10
}

Rule {
    ID: "kekw",
    Name: "test2",
    Expressions: [
        {
            Key: "key",
            Operation: "eq",
            Value: "value"
        }
    ],
    RatePerSecond: 10,
    StorageRatePerDay: 100,
    RetentionDays: 10
}

```

Будет преобразован в:

```txt
Expressions: [
    {
        Key: "key",
        Operation: "eq",
        Value: "value"
    }
]
Index: [
    [0, 1]
]
```

Тут важно, что индекс массивов становится своего рода ExpressionID, то есть массивы expressions и index можно интерпретировать как `map[ExpressionID]Expression` и `map[ExpressionID][]RuleID`.

Отдельно несколько запутавающий нейминг: RuleID и Rule.ID - это разные вещи. RuleID - это идентификатор правила в исходном массиве, а Rule.ID - это идентификатор правила в базе данных.

#### Expressions to Matchers

Expression -- это просто структура, которая описывает условие. Для того, чтобы проверять, соответствуют ли данные этой структуре, нужно преобрахзовать ее в ExpressionMatcher. Для этого все выражения вначале группируются по ключу, а затем преобразуются в массив Matchers для этого ключа в соответствии с операцией.

Пример:

```txt
Expressions: [
    {
        Key: "key",
        Operation: "=",
        Value: "value"
    },
    {
        Key: "key",
        Operation: "exists",
        Value: ""
    }
]

Matchers: {
    "key": [
        EqualValueMatcher{Values: {"value": 0}},
        ExistsMatcher{Matcher: 1}
    ]
}

```

Резутат работы matchers - это массив ExpressionID, которые соответствуют условиям.

#### Engine

Engine - это структура, которая хранит в себе все правила и выражения и индекс для быстрого поиска правил по выражениям.

```go
type EarlyEngine struct {
    expressionRules [][]RuleID // ExpressionID -> []RuleID
    expressionDone  []uint32   // ExpressionID -> bool

    rules Rules    // RuleID -> Rule
    stamp []uint32 // RuleID -> epoch
    count []uint8  // RuleID -> count

    epoch          uint32
    matchedCount   int
    matchedRuleIDs []RuleID
}
```

Когда в Engine приходит ExpressionID, он проверяет, есть ли в expressionDone этот ExpressionID. Если есть, то он пропускает его. Если нет, то он помечает его в expressionDone.

Дальше он собирает все RuleID, которые соответствуют этому ExpressionID из expressionRules.

Для каждого RuleID он проверяет, есть ли в stamp этот RuleID. Если есть, то он проверяет, что epoch совпадает с текущим epoch. Epoch - это номер текущего цикла, чтобы не очищать stamp и count каждый раз для всех правил при каждом новом логе. Если epoch не совпадает, то он очищает count и ставит текущий epoch в stamp. После чего инкрементирует count.

Если count для правила совпадает с количеством выражений в правиле, то правило считается выполненным. Оно возвращается в matchedRuleIDs.
