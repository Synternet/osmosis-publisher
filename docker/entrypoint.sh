#!/bin/sh

CMD="./osmosis-publisher"

if [ ! -z "$CA_CERT" ]; then
  CMD="$CMD --ca-cert $CA_CERT"
fi

if [ ! -z "$CLIENT_CERT" ]; then
  CMD="$CMD --client-cert $CLIENT_CERT"
fi

if [ ! -z "$CLIENT_KEY" ]; then
  CMD="$CMD --client-key $CLIENT_KEY"
fi

if [ ! -z "$DB_HOST" ]; then
  CMD="$CMD --db-host $DB_HOST"
fi

if [ ! -z "$DB_NAME" ]; then
  CMD="$CMD --db-name $DB_NAME"
fi

if [ ! -z "$DB_PASSW" ]; then
  CMD="$CMD --db-passw $DB_PASSW"
fi

if [ ! -z "$DB_PORT" ]; then
  CMD="$CMD --db-port $DB_PORT"
fi

if [ ! -z "$DB_USER" ]; then
  CMD="$CMD --db-user $DB_USER"
fi

if [ ! -z "$IDENTITY" ]; then
  CMD="$CMD --identity $IDENTITY"
fi

if [ ! -z "$NATS_URL" ]; then
  CMD="$CMD --nats-url $NATS_URL"
fi

if [ ! -z "$NATS_ACC_NKEY" ]; then
  CMD="$CMD --nats-acc-nkey $NATS_ACC_NKEY"
fi

if [ ! -z "$NATS_CREDS" ]; then
  CMD="$CMD --nats-creds $NATS_CREDS"
fi

if [ ! -z "$NATS_JWT" ]; then
  CMD="$CMD --nats-jwt $NATS_JWT"
fi

if [ ! -z "$NATS_NKEY" ]; then
  CMD="$CMD --nats-nkey $NATS_NKEY"
fi

if [ ! -z "$NATS_SUB_URL" ]; then
  CMD="$CMD --nats-sub-url $NATS_SUB_URL"
fi

if [ ! -z "$NATS_SUB_CREDS" ]; then
  CMD="$CMD --nats-sub-creds $NATS_SUB_CREDS"
fi

if [ ! -z "$NATS_SUB_JWT" ]; then
  CMD="$CMD --nats-sub-jwt $NATS_SUB_JWT"
fi

if [ ! -z "$NATS_SUB_NKEY" ]; then
  CMD="$CMD --nats-sub-nkey $NATS_SUB_NKEY"
fi

if [ ! -z "$PREFIX" ]; then
  CMD="$CMD --prefix $PREFIX"
fi

if [ ! -z "$TELEMETRY_PERIOD" ]; then
  CMD="$CMD --telemetry-period $TELEMETRY_PERIOD"
fi

if [ ! -z "$VERBOSE" ]; then
  CMD="$CMD --verbose"
fi

CMD="$CMD start "

if [ ! -z "$APP_API" ]; then
  CMD="$CMD --app-api $APP_API"
fi

if [ ! -z "$BLOCKS_TO_INDEX" ]; then
  CMD="$CMD --blocks-to-index $BLOCKS_TO_INDEX"
fi

if [ ! -z "$GRPC_API" ]; then
  CMD="$CMD --grpc-api $GRPC_API"
fi

if [ ! -z "$PUBLISHER_NAME" ]; then
  CMD="$CMD --publisher-name $PUBLISHER_NAME"
fi

if [ ! -z "$POOL_IDS" ]; then
  CMD="$CMD --pool-ids $POOL_IDS"
fi

if [ ! -z "$PRICES_SUBJECT" ]; then
  CMD="$CMD --prices-subject $PRICES_SUBJECT"
fi

if [ ! -z "$TENDERMINT_API" ]; then
  CMD="$CMD --tendermint-api $TENDERMINT_API"
fi

exec $CMD
