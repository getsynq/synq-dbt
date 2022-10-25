# synq-dbt

synq-dbt is command line tool that executes dbt and uploads dbt artifacts to [Synq](https://app.synq.io).

**Note: synq-dbt is intended to be used for dbt running on Airflow or similar. If you're dbt Cloud customer, please contact Synq for setup instructions.** 

### How does it work?

synq-dbt works in the following steps:

1) Execute your locally installed dbt. Arguments you supply synq-dbt are passes to dbt
2) Stores the exit code of the dbt command
3) Reads environment variable `SYNQ_TOKEN`
4) Uploads `manifest.json`, `run_results.json`, `catalog.json` and `schema.json` from `./target` directory to [Synq](https://app.synq.io)
4) Returns stored dbt's exit code

**Note:** synq-dbt is dbt version agnostic and works with any version of dbt.
For example, your current command `dbt run --models=finance --threads 5` becomes `synq-dbt run --models=finance --threads 5` or `dbt test --models=reports` becomes `synq-dbt test --models=reports`

### Installation

To successfully install synq-dbt you will need the following variables:

| Variable             |                              Description                             |
|----------------------|:--------------------------------------------------------------------:|
| SYNQ_TOKEN           | Synq token for your dbt integration you will receive from Synq team. |

## Getting Started

### Airflow
There are many ways to run dbt in Airflow environment, we will cover the most used setups in this guide.
In case none of these options don't work for you, please contact us, we will help you integrate synq into your Airflow.

1) In Airflow UI, go to Environment variables. Create a new ENV called `SYNQ_TOKEN` to token and paste your synq token as a value there.
2) 
   1) In case you run via a [DockerOperator](https://airflow.apache.org/docs/apache-airflow-providers-docker/stable/_api/airflow/providers/docker/operators/docker/index.html) [follow these instructions](https://github.com/getsynq/synq-dbt#airflow-with-a-docker-runner)
   2) In case you run dbt's [dbt's plugin](https://github.com/gocardless/airflow-dbt) [follow these instructions](https://github.com/getsynq/synq-dbt#airflow-with-dbt-plugin)

### Airflow with a DockerOperator

**Preflight check**:
- You have `SYNQ_TOKEN` environment variable set in Airflow UI 

**Note:** DockerOperator uses Docker container as a runner where everything is executed. For this we will need to install `synq-dbt` 
into the runner's container. 

Add the following lines to your runner's Dockerfile to install synq-dbt:

```dockerfile
ENV SYNQ_VERSION=v1.1.0
RUN wget -O /usr/bin/synq-dbt https://github.com/getsynq/synq-dbt/releases/download/${SYNQ_VERSION}/cloud-synq-dbt-${SYNQ_VERSION}-linux-amd64
RUN chmod +x /usr/bin/synq-dbt
```

Now, depending on your setup you might need to change either entrypoint (usually last line in Dockerfile) from `dbt` to `synq-dbt` 
Or in case that you change the command in the DbtOperator itself, change the command in your Airflow's Dag from `dbt` to `synq-dbt`

In the case of DbtOperator command change, the result would look similar to this:

```python
KubernetesPodOperator(
    ...
    cmds=["synq-dbt"],
    arguments=["test"],
)
```

You're all set!

### Airflow with DBT Plugin

You will need to install synq-dbt into your airflow cluster. You can [follow the instructions in the Intalling on Linux section](https://github.com/getsynq/synq-dbt#linux)


**Preflight check**:
- You have `SYNQ_TOKEN` environment variable set in Airflow UI
- You have `synq-dbt` installed in your Airflow cluster

Every `Dbt*Operator` supports a `bin` argument which dictates, what binary the operator executes.
The only change you need to do is to add this argument to all of your operators.

The result looks like:

```python
  dbt_run = DbtRunOperator(
    bin='synq-dbt',
    ...
  )
```

You're all set!

### Docker

To install a released binary to your Docker, add these lines to your Dockerfile 

```dockerfile
ENV SYNQ_VERSION=v1.1.0
RUN wget -O /usr/bin/synq-dbt https://github.com/getsynq/synq-dbt/releases/download/${SYNQ_VERSION}/cloud-synq-dbt-${SYNQ_VERSION}-linux-amd64
RUN chmod +x /usr/bin/synq-dbt
```

The `synq-dbt` command then will be available for execution.

### Linux

1) Go to [releases](https://github.com/getsynq/synq-dbt/releases) and download the latest released binary for your architecture.
2) Place the binary somewhere in your $PATH. For example `mv synq-dbt-v1.1.0 /usr/local/bin/synq-dbt` 
3) Make the binary executable `chmod +x /usr/local/bin/synq-dbt`