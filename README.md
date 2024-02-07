# Osmosis Publisher
[![Latest release](https://img.shields.io/github/v/release/SyntropyNet/osmosis-publisher)](https://github.com/SyntropyNet/osmosis-publisher/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GitHub Workflow Status (with event)](https://img.shields.io/github/actions/workflow/status/SyntropyNet/osmosis-publisher/github-ci.yml?label=github-ci)](https://github.com/SyntropyNet/osmosis-publisher/actions/workflows/github-ci.yml)

Establishes connection with Osmosis node and publishes Osmosis blockchain data to Syntropy Data Layer via NATS connection.

# Usage

Building from source.
```bash
make build
```

Getting usage help.
```bash
./build/osmosis-publisher --help
```

Running executable with flags.
```bash
./build/osmosis-publisher \
  --nats-url nats://dal-broker \
  --prefix my-org \
  --nats-nkey SA..BC \
  --nats-jwt eyJ0e...aW \
  --db-host db.sqlite \
  --db-name sqlite
  start \
  --app-api http://localhost:1317 \
  --grpc-api localhost:9090 \
  --tendermint-api tcp://localhost:26657 \
  --publisher-name osmosis
```

Running executable with environment variables. Environment variables are automatically attempted to be loaded from `.env` file.
Any flag can be used as environment variables by updating flag to be `UPPERCASE` words separated by `_` (e.g.: flag `nats-nkey` == env var `NATS_NKEY`).
```
./build/osmosis-publisher start

// .env file content
NATS_URL=nats://dal-broker
PREFIX=my-org
NATS_NKEY=SA..BC
NATS_JWT=eyJ0e...aW
DB_HOST=db.sqlite
DB_NAME=sqlite
APP_API=http://localhost:1317
GRPC_API=localhost:9090
TENDERMINT_API=tcp://localhost:26657
PUBLISHER_NAME=osmosis
```

Note: instead of user `NATS_NKEY` and `NATS_JWT` single value of `NATS_ACC_NKEY` can be supplied. In Syntropy Data Layer Developer Portal
this is called `Access Token`. See [here](https://docs.syntropynet.com/build/data-layer/developer-portal/data-layer-authentication#access-token) for more details.

## Docker

### Build from source

1. Build image.
```
docker build -f ./docker/Dockerfile -t osmosis-publisher .
```

2. Run container with passed environment variables. See [entrypoint.sh](./docker/entrypoint.sh) for available env variables in container.
```
docker run -it --rm --env-file=.env osmosis-publisher
```

### Prebuilt image

Run container with passed environment variables.
```
docker run -it --rm --env-file=.env ghcr.io/syntropynet/osmosis-publisher:latest
```

### Docker Compose

`docker-compose.yml` file.
```
version: '3.8'

services:
  osmosis-publisher:
    image: ghcr.io/syntropynet/osmosis-publisher:latest
    environment:
      - NATS_URL=nats://dal-broker
      - PREFIX=my-org
      - NATS_NKEY=SA..BC
      - NATS_JWT=eyJ0e...aW
      - APP_API=http://localhost:1317
      - GRPC_API=localhost:9090
      - TENDERMINT_API=tcp://localhost:26657
      - PUBLISHER_NAME=osmosis
      - DB_HOST=db.sqlite
      - DB_NAME=sqlite
    volumes:
      - osmodata:/home/app/postgres:ro
volumes:
  osmodata:
    external: true
```

## Contributing

We welcome contributions from the community. Whether it's a bug report, a new feature, or a code fix, your input is valued and appreciated.

## Syntropy

If you have any questions, ideas, or simply want to connect with us, we encourage you to reach out through any of the following channels:

- **Discord**: Join our vibrant community on Discord at [https://discord.com/invite/jqZur5S3KZ](https://discord.com/invite/jqZur5S3KZ). Engage in discussions, seek assistance, and collaborate with like-minded individuals.
- **Telegram**: Connect with us on Telegram at [https://t.me/SyntropyNet](https://t.me/SyntropyNet). Stay updated with the latest news, announcements, and interact with our team members and community.
- **Email**: If you prefer email communication, feel free to reach out to us at devrel@syntropynet.com. We're here to address your inquiries, provide support, and explore collaboration opportunities.
