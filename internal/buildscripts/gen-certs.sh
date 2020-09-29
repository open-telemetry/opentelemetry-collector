#!/usr/bin/env bash

# This script is used to create the CA, server and client's certificates and keys required by unit tests.
# These certificates use the Subject Alternative Name extension rather than the Common Name, which will be unsupported from Go 1.15.

usage() {
  echo "Usage: $0 [-d]"
  echo
  echo "-d  Dry-run mode. No project files will not be modified. Default: 'false'"
  echo "-m  Domain name to use in the certificate. Default: 'localhost'"
  echo "-o  Output directory where certificates will be written to. Default: '.'; the current directory"
  exit 1
}

dry_run=false
domain="localhost"
output_dir="."

while getopts "dm:o:" o; do
    case "${o}" in
        d)
            dry_run=true
            ;;
        m)
            domain=$OPTARG
            ;;
        o)
            output_dir=$OPTARG
            ;;
        *)
            usage
            ;;
    esac
done
shift $((OPTIND-1))

set -ex

# Create temp dir for generated files.
tmp_dir=$(mktemp -d -t certificates)
clean_up() {
    ARG=$?
    if [ $dry_run = true ]; then
      echo "Dry-run complete. Generated files can be found in $tmp_dir"
    else
      rm -rf "$tmp_dir"
    fi
    exit $ARG
}
trap clean_up EXIT

gen_ssl_conf() {
  domain_name=$1
  output_file=$2

  cat << EOF > "$output_file"
[ req ]
prompt              = no
default_bits        = 2048
distinguished_name  = req_distinguished_name
req_extensions      = req_ext

[ req_distinguished_name ]
countryName         = AU
stateOrProvinceName = Australia
localityName        = Sydney
organizationName    = MyOrgName
commonName          = MyCommonName

[ req_ext ]
subjectAltName      = @alt_names

[alt_names]
DNS.1               = $domain_name
EOF
}

# Generate config files.
gen_ssl_conf "$domain" "$tmp_dir/ssl.conf"

# Create CA (accept defaults from prompts).
openssl genrsa -out "$tmp_dir/ca.key"  2048
openssl req -new -key "$tmp_dir/ca.key" -x509 -days 3650 -out "$tmp_dir/ca.crt" -config "$tmp_dir/ssl.conf"

# Create client and server keys.
openssl genrsa -out "$tmp_dir/server.key" 2048
openssl genrsa -out "$tmp_dir/client.key" 2048

# Create certificate sign request using the above created keys.
openssl req -new -nodes -key "$tmp_dir/server.key" -out "$tmp_dir/server.csr" -config "$tmp_dir/ssl.conf"
openssl req -new -nodes -key "$tmp_dir/client.key" -out "$tmp_dir/client.csr" -config "$tmp_dir/ssl.conf"

# Creating the client and server certificates.
openssl x509 -req \
             -sha256 \
             -days 3650 \
             -in "$tmp_dir/server.csr" \
             -signkey "$tmp_dir/server.key" \
             -out "$tmp_dir/server.crt" \
             -extensions req_ext \
             -CA "$tmp_dir/ca.crt" \
             -CAkey "$tmp_dir/ca.key" \
             -CAcreateserial \
             -extfile "$tmp_dir/ssl.conf"
openssl x509 -req \
             -sha256 \
             -days 3650 \
             -in "$tmp_dir/client.csr" \
             -signkey "$tmp_dir/client.key" \
             -out "$tmp_dir/client.crt" \
             -extensions req_ext \
             -CA "$tmp_dir/ca.crt" \
             -CAkey "$tmp_dir/ca.key" \
             -CAcreateserial \
             -extfile "$tmp_dir/ssl.conf"

# Copy files if not in dry-run mode.
if [ $dry_run = false ]; then
  cp "$tmp_dir/ca.crt" \
     "$tmp_dir/client.crt" \
     "$tmp_dir/client.key" \
     "$tmp_dir/server.crt" \
     "$tmp_dir/server.key" \
     "$output_dir"
fi
