# synq-dbt

synq-dbt is command line tool that executes dbt and uploads dbt artifacts to [Synq](https://app.synq.io).

**Note: synq-dbt is intended to be used for dbt running on Airflow or similar. If you're dbt Cloud customer, please contact Synq for setup instructions.** 

### How does it work?

synq-dbt is dbt version agnostic and works with any version of dbt in the following steps:

1) Execute your locally installed `dbt`. Arguments you supply to `synq-dbt` are passes to `dbt`. For example, your current command `dbt run --models=finance --threads 5` becomes `synq-dbt run --models=finance --threads 5` or `dbt test --models=reports` becomes `synq-dbt test --models=reports`.
2) Stores the exit code of the dbt command.
3) Reads environment variable `SYNQ_TOKEN`.
4) Uploads `manifest.json`, `run_results.json`, `catalog.json` and `schema.json` from `./target` directory to [Synq](https://app.synq.io).
4) Returns stored dbt's exit code.



## Installation

To successfully install synq-dbt you will need `SYNQ_TOKEN` secret, that you receive from Synq team. It should be treated as a secret as it allows Synq to identify you as customer to associate uploaded data with your workspace.

### Airflow

We will cover two most common setups of dbt and Airflow. In case none of these options works for you, please contact us.

1) In case you run dbt via a [DockerOperator](https://airflow.apache.org/docs/apache-airflow-providers-docker/stable/_api/airflow/providers/docker/operators/docker/index.html) [follow these instructions](https://github.com/getsynq/synq-dbt#airflow-with-a-docker-runner)
2) In case you run dbt via [dbt's plugin](https://github.com/gocardless/airflow-dbt) [follow these instructions](https://github.com/getsynq/synq-dbt#airflow-with-dbt-plugin)

### Airflow with a DockerOperator

1) In Airflow UI, go to Environment variables. Create a new ENV variable called `SYNQ_TOKEN` with Synq token as a value.

2) Install `synq-dbt` into Docker container that is executed by your DockerOperator

Add the following lines to your runner's Dockerfile to install synq-dbt:

```dockerfile
ENV SYNQ_VERSION=v1.2.2
RUN wget -O /usr/bin/synq-dbt https://github.com/getsynq/synq-dbt/releases/download/${SYNQ_VERSION}/synq-dbt-amd64-linux
RUN chmod +x /usr/bin/synq-dbt
```

3) Change Docker container entrypoint (usually last line in Dockerfile) from `dbt` to `synq-dbt` OR change the command in the DbtOperator itself in your Airflow's Dag from `dbt` to `synq-dbt`

In the case of DbtOperator command change, the result would look as follows:

```python
KubernetesPodOperator(
    ...
    cmds=["synq-dbt"],
    arguments=["test"],
)
```

You're all set! :tada:

### Airflow with DBT Plugin

1) In Airflow UI, go to Environment variables. Create a new ENV variable called `SYNQ_TOKEN` with Synq token as a value.

2) Install synq-dbt into your airflow cluster. You can [follow the instructions in the Intalling on Linux section](https://github.com/getsynq/synq-dbt#linux)

3) Every `Dbt*Operator` supports a `bin` argument which specifies, what binary the operator executes.

The result should look as follows:

```python
  dbt_run = DbtRunOperator(
    bin='synq-dbt',
    ...
  )
```

You're all set! :tada:

### Docker

To install a released binary to your Docker, add these lines to your Dockerfile 

```dockerfile
ENV SYNQ_VERSION=v1.2.2
RUN wget -O /usr/bin/synq-dbt https://github.com/getsynq/synq-dbt/releases/download/${SYNQ_VERSION}/synq-dbt-amd64-linux
RUN chmod +x /usr/bin/synq-dbt
```

The `synq-dbt` command then will be available for execution.

### Linux

1) Go to [releases](https://github.com/getsynq/synq-dbt/releases) and download the latest released binary for your architecture.
2) Place the binary in your $PATH. For example `mv synq-dbt-v1.2.2 /usr/local/bin/synq-dbt` 
3) Make the binary executable `chmod +x /usr/local/bin/synq-dbt`