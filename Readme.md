# Synq dbt wrapper


The present repository contains the source code of the [Synq](https://app.synq.io) dbt wrapper.
Dbt wrapper is a simple command line tool, that executes dbt and uploads the dbt's manifests to synq.

**Note: This wrapper is only used for locally (Airflow and similar) run dbt. In case you run dbt using DbtCloud please contact us for setup instructions.** 

## Documentation

The general documentation of the project, including instructions for installation.

## Arguments

| Environment Variable |                                                   Description                                                   |
|----------------------|:---------------------------------------------------------------------------------------------------------------:|
| SYNQ_TOKEN           | Synq token for your dbt integration. You should have already gotten it from our Team. If not, please contact us |

## Getting Started

The command line wrapper is a simple tool, that:
1) Executes your locally installed dbt and passes all the arguments into the dbt command
2) Stores the exit code of the dbt command
3) Reads env `SYNQ_TOKEN` and uploads your `manifest.json`, `run_results.json`, `catalog.json` and `schema.json` from `./target` directory to [Synq](https://app.synq.io)
4) Returns stored dbt's exit code (Even if the upload fails)

**Note:** This wrapper is dbt version agnostic. Update dbt as you would normally. The wrapper only executes your dbt as you would in shell.
In practice, make an ENV called `SYNQ_TOKEN` then your dbt command is just interchanged from `dbt run --models=finance --threads 5` and/or `dbt test --models=reports` to `synq_dbt run --models=finance --threads 5` and/or `synq_dbt test --models=reports`

### Installation

### Airflow
There are many ways to run dbt in Airflow environment, we will cover the most used setups in this guide.
In case none of these options don't work for you, please contact us, we will help you integrate synq into your Airflow.

1) In Airflow UI, go to Environment variables. Create a new ENV called `SYNQ_TOKEN` to token and paste your synq token as a value there.
2) Depending on your airflow setup, you likely either use [dbt's plugin](https://github.com/gocardless/airflow-dbt) and run that as an DBT*Operator or [DockerOperator](https://airflow.apache.org/docs/apache-airflow-providers-docker/stable/_api/airflow/providers/docker/operators/docker/index.html)
   - In case you run your dbt inside a DockerOperator [follow these instructions](https://github.com/getsynq/dbt-wrapper#airflow-with-a-docker-runner)
   - In case you run dbt's airflow plugin [follow these instructions](https://github.com/getsynq/dbt-wrapper#airflow-with-dbt-plugin)

### Airflow with a Docker Runner

**Preflight check**:
- You have `SYNQ_TOKEN` environment variable set in Airflow UI 

**Note:** DockerOperator uses Docker container as a runner where everything is executed. For this we will need to install `synq_dbt` 
into the runner's container. 

Simply add these lines to your runner's Dockerfile:

```dockerfile
ENV SYNQ_VERSION=v1.1.0
RUN wget -O /usr/bin/synq_dbt https://github.com/getsynq/dbt-wrapper/releases/download/${SYNQ_VERSION}/cloud-synq-dbt-${SYNQ_VERSION}-linux-amd64
RUN chmod +x /usr/bin/synq_dbt
```

Now, depending on your setup you might need to change either entrypoint (usually last line in Dockerfile) from `dbt` to `synq_dbt` 
Or in case that you change the command in the DbtOperator itself, change the command in your Airflow's Dag from `dbt` to `synq_dbt`

In the case of DbtOperator command change, the result would look similar to this:

```python
KubernetesPodOperator(
    ...
    cmds=["synq_dbt"],
    arguments=["test"],
)
```

You're all set!

### Airflow with DBT Plugin

You will need to install the wrapper into your airflow cluster. You can [follow the instructions in the Intalling on Linux section](https://github.com/getsynq/dbt-wrapper#linux)


**Preflight check**:
- You have `SYNQ_TOKEN` environment variable set in Airflow UI
- You have `synq_dbt` installed in your Airflow cluster

Every `Dbt*Operator` supports a `bin` argument which dictates, what binary the operator executes.
The only change you need to do is to add this argument to all of your operators.

The result looks like:

```python
  dbt_run = DbtRunOperator(
    bin='synq_dbt',
    ...
  )
```

You're all set!

### Docker

To install a released binary to your Docker, add these lines to your Dockerfile 

```dockerfile
ENV SYNQ_VERSION=v1.1.0
RUN wget -O /usr/bin/synq_dbt https://github.com/getsynq/dbt-wrapper/releases/download/${SYNQ_VERSION}/cloud-synq-dbt-${SYNQ_VERSION}-linux-amd64
RUN chmod +x /usr/bin/synq_dbt
```

The `synq_dbt` command then will be available for execution.

### Linux

1) Go to [releases](https://github.com/getsynq/dbt-wrapper/releases) and download the latest released binary for your architecture.
2) Place the binary somewhere in your $PATH. For example `mv synq-dbt-v1.1.0 /usr/local/bin/synq_dbt` 
3) Make the binary executable `chmod +x /usr/local/bin/synq_dbt`