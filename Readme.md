# synq-dbt

synq-dbt is a command line tool that executes dbt and uploads dbt artifacts to [SYNQ](https://www.synq.io).

**Note: synq-dbt is intended to be used for dbt running on Airflow or similar. If you're a dbt Cloud customer, you can integrate you account within SYNQ by going to Settings -> Integrations -> Add Integration -> dbt Cloud.**

# Wrapping dbt execution

`synq-dbt` wraps `dbt` command. After execution of `dbt` it collects [dbt artifacts](https://docs.getdbt.com/reference/artifacts/dbt-artifacts) that allow SYNQ to understand the structure and status of your dbt project. We collect the following:

- `manifest.json` — to understand the structure of the dbt project
- `run_results.json` — to understand executions status
- `catalog.json` — to infer the complete schema of underlying data warehouse tables
- `sources.json` — to capture dbt source freshness

To control the location of artifacts, you can use the `SYNQ_TARGET_DIR` environment variable. By default, `synq-dbt` looks for `target/` directory in the current working directory.

All the data is presented in the [SYNQ](https://www.synq.io).

`synq-dbt` is dbt version agnostic and works with the version of dbt you have installed on your system. It runs in the following steps:

1) Execute your locally installed `dbt`. Arguments you supply to `synq-dbt` are passed to `dbt`. For example, your current command `dbt run --select finance --threads 5` becomes `synq-dbt run --select finance --threads 5` or `dbt test --select reports` becomes `synq-dbt test --select reports`.
2) Stores the exit code of the dbt command.
3) Reads environment variable `SYNQ_TOKEN`.
4) Uploads `manifest.json`, `run_results.json`, `catalog.json` and `schema.json` from `./target` directory to [SYNQ](https://www.synq.io).
5) Returns stored dbt's exit code. synq-dbt ignores its own errors and always exists with error code of dbt subcommand.

# Uploading already existent artifacts

It is possible to upload artifacts that have already been generated. In that case, you can use `synq-dbt synq_upload_artifacts` command to upload artifacts to SYNQ.

```shell

export SYNQ_VERSION=v1.7.0
wget -O ./synq-dbt https://github.com/getsynq/synq-dbt/releases/download/${SYNQ_VERSION}/synq-dbt-amd64-linux
chmod +x ./synq-dbt

export SYNQ_TOKEN=<your-token>
./synq-dbt synq_upload_artifacts
```

It is possible to include in the uploaded request logs of dbt execution, to do that you need to generate dbt logs to a file and then point synq-dbt to that file.

```shell
dbt build | tee dbt.log
./synq-dbt synq_upload_artifacts --dbt-log-file dbt.log

```

# Installation

To successfully install and launch `synq-dbt` you will need `SYNQ_TOKEN` secret, that you generate in your SYNQ account when integrating with dbt Core. Reach out to the team if you have any questions. It should be treated as a secret as it allows SYNQ to identify you as the customer and associate uploaded data with your workspace.

## Airflow

We will cover two most common setups of dbt and Airflow:

1) [DockerOperator](https://airflow.apache.org/docs/apache-airflow-providers-docker/stable/_api/airflow/providers/docker/operators/docker/index.html) => [follow these instructions](https://github.com/getsynq/synq-dbt#airflow-with-a-docker-runner)
2) [dbt's plugin](https://github.com/gocardless/airflow-dbt) => [follow these instructions](https://github.com/getsynq/synq-dbt#airflow-with-dbt-plugin)

In case none of these works for you, don't hesitate to get in touch with us.

### Airflow with a DockerOperator

1) In Airflow UI, go to Environment variables. Create a new ENV variable called `SYNQ_TOKEN` with SYNQ token as a value.

2) Install `synq-dbt` into Docker container that is executed by your DockerOperator

Add the following lines to your runner's Dockerfile to install `synq-dbt`:

```dockerfile
ENV SYNQ_VERSION=v1.5.0
RUN wget -O /usr/bin/synq-dbt https://github.com/getsynq/synq-dbt/releases/download/${SYNQ_VERSION}/synq-dbt-amd64-linux
RUN chmod +x /usr/bin/synq-dbt
```

3) Change Docker container entrypoint (usually last line in Dockerfile) from `dbt` to `synq-dbt` OR change the command in the DbtOperator itself in your Airflow's Dag from `dbt` to `synq-dbt`

In the case of `KubernetesPodOperator` change, the result should for example look as follows:

```python
KubernetesPodOperator(
    ...
    env_vars={
        "SYNQ_TOKEN": Variable.get("SYNQ_TOKEN")
    },
    cmds=["synq-dbt"],
    arguments=["test"],
)
```

You're all set! :tada:

### Airflow with dbt Plugin

1) In Airflow UI, go to Environment variables. Create a new ENV variable called `SYNQ_TOKEN` with SYNQ token as a value.

2) Execute the following shell commands to download the latest version of `synq-dbt`

```shell
export SYNQ_VERSION=v1.5.0
wget -O ./synq-dbt https://github.com/getsynq/synq-dbt/releases/download/${SYNQ_VERSION}/synq-dbt-amd64-linux
chmod +x ./synq-dbt
```

3) Move the `synq-dbt` binary in your $PATH

```shell
mv synq-dbt /usr/local/bin/synq-dbt
```

3) Unfortunatelly, Dbt*Operators haven't been released to pip for quite some time and [a pull-request that added **env** argument was not released to pip yet](https://github.com/gocardless/airflow-dbt/pull/60)
   In case you build the airflow-dbt locally
   Change your `Dbt*Operator`s `dbt_bin` argument as follows:

```python
  dbt_run = DbtRunOperator(
    env={
        "SYNQ_TOKEN": Variable.get("SYNQ_TOKEN")
    },
    dbt_bin='synq-dbt',
    ...
  )
```

Otherwise, you will need to set an env `SYNQ_TOKEN` on the system running Airflow with:
```
export SYNQ_TOKEN=your-synq-token
```

You're all set! :tada:

## Docker

Add the following lines to your Dockerfile:

```dockerfile
ENV SYNQ_VERSION=v1.5.0
RUN wget -O /usr/bin/synq-dbt https://github.com/getsynq/synq-dbt/releases/download/${SYNQ_VERSION}/synq-dbt-amd64-linux
RUN chmod +x /usr/bin/synq-dbt
```

The `synq-dbt` command is available for execution. :tada:

## Linux

1) Execute the following shell commands to download the latest version of `synq-dbt`

```shell
export SYNQ_VERSION=v1.5.0
wget -O ./synq-dbt https://github.com/getsynq/synq-dbt/releases/download/${SYNQ_VERSION}/synq-dbt-amd64-linux
chmod +x ./synq-dbt
```

2) Move the `synq-dbt` binary in your $PATH

```shell
mv synq-dbt /usr/local/bin/synq-dbt
```

The `synq-dbt` command is available for execution. :tada:

## OSX

OSX version is primarily used for testing, by manually triggering `synq-ctl`.

1) Execute the following shell commands to download the latest version of `synq-dbt`

```shell
export SYNQ_VERSION=v1.5.0
wget -O ./synq-dbt https://github.com/getsynq/synq-dbt/releases/download/${SYNQ_VERSION}/synq-dbt-arm64-darwin
chmod +x ./synq-dbt
```

2) Move the `synq-dbt` binary in your $PATH

```shell
mv synq-dbt /usr/local/bin/synq-dbt
```

3) Export your `SYNQ_TOKEN` to the current shell

```shell
export SYNQ_TOKEN=<your-token>
```


4) Check current version of `dbt` via `synq-dbt` as follows:

```shell
synq-dbt --version
```

will result in:

```shell
07:04:54  synq-dbt processing `dbt --version`
Core:
  - installed: 1.2.0
  - latest:    1.3.0 - Update available!

...
```

You're all set! :tada:

**Note: Note when testing `synq-dbt` locally on your mac, it is recommended you delete `target/` folder before you execute `synq-dbt` so it doesn't pickup old dbt artifacts.**

## Dagster
1) In the `.env` file in your root directory, create a variable called `SYNQ_TOKEN` with SYNQ token as a value (i.e. `SYNQ_TOKEN=<TOKEN_VALUE>`).
2) In your `definitions.py` file, update your dbt resources definition to use `synq-dbt`

```python
resources={
        "dbt": DbtCliResource(dbt_executable='synq-dbt', project_dir=os.fspath(dbt_project_dir)),
}
```
3) By default, Dagster creates a dynamic path for the dbt artifacts but SYNQ always looks in the root target folder. In your `assets.py` file, update the `target_path` so that artifacts are stored in the root target folder

```python
@dbt_assets(manifest=dbt_manifest_path)
def jaffle_shop_dbt_assets(context: AssetExecutionContext, dbt: DbtCliResource):
    dbt_target_path = Path('target')
    yield from dbt.cli(["build"], target_path=dbt_target_path, context=context).stream()
```


# FAQ

### **Q:** What version of `dbt` does `synq-dbt` run?

**A:** `synq-dbt` is `dbt` version agnostic and **works with the version of dbt you have installed on your system**.

##



### **Q:** What requests does `synq-dbt` do?

**A:** Every time it executes `synq-dbt,` one gRPC request is made to SYNQ servers. The payload of the request contains dbt artifacts and an authentication token that the server uses to verify your data.

**Note: Depending on your setup, you might have to allow egress traffic in your network firewall to `dbtapi.synq.io:443`.**

##



### **Q:** What is the size of the payload?

**A:** Since most of the data is text, the total size of the payload is roughly equivalent to the sum of the sizes of dbt artifacts. `dbt_manifest.json` is usually the largest, and the final size of the request depends on the size of your project, ranging from a few MBs to higher tens of MBs typically.

**Note: Depending on your setup, you might have to allow large payloads in your network firewall.**

##



### **Q:** How quickly does data appear in SYNQ UI?

**A:** Unless our system experiences an unusual traffic spike, data should be available in UI within a few minutes.
