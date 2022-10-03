
# Eth2-validator

Eth2-validator (placeholder name) is a lightweight Ethereum validator.

## Usage

Deploy a consensus devnet with [Viewpoint](https://github.com/umbracle/viewpoint).

Run the `server` command to start the `Viewpoint` server:

```
$ viewpoint server --name test-altair-1 --num-tranches 2 --altair 1
```

In another terminal, deploy a `Lighthouse` validator and two beacon nodes assigned to half of the genesis validator accounts:

```
$ viewpoint node deploy validator --type lighthouse --tranche 0 --beacon --beacon-count 2
```

Run `Eth2-validator`:

```
$ eth2-validator server --endpoint http://127.0.0.1:<port> --priv-key ./e2e-test-altair-1/tranche_1.txt --log-level debug
```

where `port` is the Http port of one of the beacon nodes deployed in the previous step.
