# Memory Ballast

Memory Ballast extension enables applications to configure memory ballast for the process. For more details see:
- [Go memory ballast blogpost](https://blog.twitch.tv/go-memory-ballast-how-i-learnt-to-stop-worrying-and-love-the-heap-26c2462549a2)
- [Golang issue related to this](https://github.com/golang/go/issues/23044)

The following settings can be configured:

- `size_mib` (default = 0, disabled): Is the memory ballast size, in MiB.

Example:

```yaml
extensions:
  memory_ballast:
    size_mib: 64
```
