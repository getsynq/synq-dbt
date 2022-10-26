# synq-dbt

synq-dbt is command line tool that executes dbt and uploads dbt artifacts to [Synq](https://app.synq.io).

**Note: synq-dbt is intended to be used for dbt running on Airflow or similar. If you're dbt Cloud customer, please contact Synq for setup instructions.** 

# How does it work?

synq-dbt is dbt version agnostic and works with any version of dbt in the following steps:

1) Execute your locally installed `dbt`. Arguments you supply to `synq-dbt` are passes to `dbt`. For example, your current command `dbt run --select finance --threads 5` becomes `synq-dbt run --select finance --threads 5` or `dbt test --select reports` becomes `synq-dbt test --select reports`.
2) Stores the exit code of the dbt command.
3) Reads environment variable `SYNQ_TOKEN`.
4) Uploads `manifest.json`, `run_results.json`, `catalog.json` and `schema.json` from `./target` directory to [Synq](https://app.synq.io).
4) Returns stored dbt's exit code.

# Installation

To successfully install `synq-dbt` you will need `SYNQ_TOKEN` secret, that you receive from Synq team. It should be treated as a secret as it allows Synq to identify you as customer to associate uploaded data with your workspace.

## Airflow

We will cover two most common setups of dbt and Airflow:

1) [DockerOperator](https://airflow.apache.org/docs/apache-airflow-providers-docker/stable/_api/airflow/providers/docker/operators/docker/index.html) => [follow these instructions](https://github.com/getsynq/synq-dbt#airflow-with-a-docker-runner)
2) [dbt's plugin](https://github.com/gocardless/airflow-dbt) => [follow these instructions](https://github.com/getsynq/synq-dbt#airflow-with-dbt-plugin)

In case none of these works for you, please contact us.

### Airflow with a DockerOperator

1) In Airflow UI, go to Environment variables. Create a new ENV variable called `SYNQ_TOKEN` with Synq token as a value.

2) Install `synq-dbt` into Docker container that is executed by your DockerOperator

Add the following lines to your runner's Dockerfile to install `synq-dbt`:

```dockerfile
ENV SYNQ_VERSION=v1.2.2
RUN wget -O /usr/bin/synq-dbt https://github.com/getsynq/synq-dbt/releases/download/${SYNQ_VERSION}/synq-dbt-amd64-linux
RUN chmod +x /usr/bin/synq-dbt
```

3) Change Docker container entrypoint (usually last line in Dockerfile) from `dbt` to `synq-dbt` OR change the command in the DbtOperator itself in your Airflow's Dag from `dbt` to `synq-dbt`

In the case of `KubernetesPodOperator` change, the result should for example look as follows:

```python
KubernetesPodOperator(
    ...
    cmds=["synq-dbt"],
    arguments=["test"],
)
```

You're all set! :tada:

### Airflow with dbt Plugin

1) In Airflow UI, go to Environment variables. Create a new ENV variable called `SYNQ_TOKEN` with Synq token as a value.

2) Execute the following shell commands to download the latest version of `synq-dbt`

```console
export SYNQ_VERSION=v1.2.2
wget -O ./synq-dbt https://github.com/getsynq/synq-dbt/releases/download/${SYNQ_VERSION}/synq-dbt-amd64-linux
chmod +x ./synq-dbt
```

3) Move the `synq-dbt` binary in your $PATH

```console
mv synq-dbt /usr/local/bin/synq-dbt
```

3) Change your `Dbt*Operator`s `bin` argument as follows:

```python
  dbt_run = DbtRunOperator(
    bin='synq-dbt',
    ...
  )
```

You're all set! :tada:

## Docker

Add the following lines to your Dockerfile:

```dockerfile
ENV SYNQ_VERSION=v1.2.2
RUN wget -O /usr/bin/synq-dbt https://github.com/getsynq/synq-dbt/releases/download/${SYNQ_VERSION}/synq-dbt-amd64-linux
RUN chmod +x /usr/bin/synq-dbt
```

The `synq-dbt` command is available for execution. :tada:

## Linux

1) Execute the following shell commands to download the latest version of `synq-dbt`

```console
export SYNQ_VERSION=v1.2.2
wget -O ./synq-dbt https://github.com/getsynq/synq-dbt/releases/download/${SYNQ_VERSION}/synq-dbt-amd64-linux
chmod +x ./synq-dbt
```

2) Move the `synq-dbt` binary in your $PATH

```console
mv synq-dbt /usr/local/bin/synq-dbt
```

The `synq-dbt` command is available for execution. :tada:

## OSX

OSX version is primarily used for testing, by manually triggering `synq-ctl`. You can use it 

1) Execute the following shell commands to download the latest version of `synq-dbt`

```console
export SYNQ_VERSION=v1.2.2
wget -O ./synq-dbt https://github.com/getsynq/synq-dbt/releases/download/${SYNQ_VERSION}/synq-dbt-arm64-darwin
chmod +x ./synq-dbt
```

2) Move the `synq-dbt` binary in your $PATH

```console
mv synq-dbt /usr/local/bin/synq-dbt
```

3) Check current version of `dbt` via `synq-dbt` as follows:

```console
$ synq-dbt --version
```

will result in:

```console
07:04:54  synq-dbt failed: missing SYNQ_TOKEN variable
07:04:54  synq-dbt processing `dbt --version`
Core:
  - installed: 1.2.0
  - latest:    1.3.0 - Update available!

...
```

You're all set! :tada:

# FAQ

**Q:** What requests does `synq-dbt` do?

**A:** Every time it executes `synq-dbt` does one gRPC request to Synq servers. The payload of the request contains dbt artifacts and authentication token that server uses to verify your data.

**Note: Depending on your setup, you might have to allow egress traffic in your network firewall to `dbt-uploader-xwpzuoapgq-lm.a.run.app:443`.**

##

**Q:** What is the size of the payload?

**A:** Since most of the data is text, total size of payload is roughly equivalent to sum of sizes of dbt artifacts. `dbt_manifest.json` is usually the largest and the final size of the request depends on size of your project, ranging from few MBs to higher tens of MBs typically.

**Note: Depending on your setup, you might have to allow large payloads in your network firewall.**

##

**Q:** How quickly does data appear in Synq UI?

**A:** Unless our system experiences unusual traffic spike, data should be available in UI within a few minutes.