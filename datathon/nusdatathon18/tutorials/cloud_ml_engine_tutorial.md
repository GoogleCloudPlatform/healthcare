# Cloud ML Engine Tutorial

[Google Cloud Machine Learning (ML) Engine](https://cloud.google.com/ml-engine/) is a managed service that enables developers and data scientists to build and bring superior machine learning models to production.

In this tutorial, we are going to go further beyond what we did in [the machine learning colab](ddsm_ml_tutorial.ipynb)(please check out the colab first if you haven't), and use Cloud ML Engine to train a simple CNN with both dedicated GPU and TPU to classify CBIS-DDSM images based on breast density.

## Setup

Please follow the instructions below to set up the environment required to use Cloud ML Engine.

* Follow [the official instructions](https://cloud.google.com/sdk/install) to install Google Cloud SDK to your computer, or click [here](https://console.cloud.google.com/home/dashboard?cloudshell=true) to access a browser-based shell with Google Cloud SDK installed.
* Next, download the GPU and TPU models from Google Cloud Storage by running the following command.

```shell
gsutil cp -r gs://cbis-ddsm-colab/cbis_ddsm_ml .
```

## Training

Let's first start training with GPU by running the following command in the Google Cloud SDK shell.

```shell
gcloud ml-engine jobs submit training gpu_training \
    --staging-bucket gs://cbis-ddsm-artifacts \
    --runtime-version 1.8 \
    --scale-tier BASIC_GPU \
    --module-name "trainer.gpu_model" \
    --package-path cbis_ddsm_ml/trainer \
    --region us-central1 \
    -- \
    --image_width=95 \
    --image_height=128 \
    --training_data_dir="small_train" \
    --eval_data_dir="small_test" \
    --model_dir="gs://cbis-ddsm-model/`date +%s`" \
    --category_count=4 \
    --data_bucket="cbis-ddsm-colab" \
    --training_steps=1000
```

That's a long list of parameters. Let's go through one by one.

* "gpu_training" is the name of the job and needs to be unique within the same project, you can change it to something else if you want or the name is already taken.

Then we have a few parameters for Cloud ML Engine, normally you don't need to modify these (but please use your team's shared buckets during the datathon for --staging-bucket and --model_dir).

* `--staging-bucket` specifies where to store the intermediate state for training.
* `--runtime-version` specifies that we want to use Tensowflow 1.8 which is the latest version as of June 2018.
* `--scale-tier` is set to BASIC_GPU to use GPU for training.
* `--module-name` and `--package-path` tell Cloud ML Engine where to look for our training code and its structure.
  * You might need to adjust `--package-path` depending on where the models are extracted during setup.
* `--region` specifies in which region our training job is run. Right now TPUs are only available in `us-central1`, we train with GPU in the same region for easier comparison and access control.

At the bottom we have customized parameters for our model:

* `--image_width` and `--image_height` are dimensions of our input images which are preprocessed beforehand.
* `--data_bucket`, `--training_data_dir` and `--eval_data_dir` are locations of our input data in GCS.
* `--model_dir` is used to store intermediate states and trained model.
* `--category_count` is the final number of categories to classify the images into. In our case, breast density is an integer that falls in [1, 4].
* `--training_steps` specifies how many steps the model will train for.

There are a few other flags that are not covered, please feel free to take a look at the model code and figure out their usage.

After the job is submitted, Cloud ML Engine will take care of acquiring the required resources and dispatching training tasks. Meanwhile, you can query the status by running:

```shell
# gpu_training is the name of the submitted job.
gcloud ml-engine jobs describe gpu_training
```

Notice that at the end of the output, there are links to both the UI and logs of the job. The whole training process may take some time (around 25 mins) to finish. If you need to reschedule the job for some reason, please first cancel the existing job to free up the resources.

Now let's proceed to submit another job which uses TPU for training while waiting for GPU training to finish. Just like GPU training, run the following command in the Google Cloud SDK shell.

```shell
gcloud ml-engine jobs submit training tpu_training \
    --staging-bucket gs://cbis-ddsm-artifacts \
    --runtime-version 1.8 \
    --scale-tier BASIC_TPU \
    --module-name "trainer.tpu_model" \
    --package-path cbis_ddsm_ml/trainer \
    --region us-central1 \
    -- \
    --image_width=95 \
    --image_height=128 \
    --training_data="gs://cbis-ddsm-colab/cache/ddsm_train.tfrecords" \
    --eval_data="gs://cbis-ddsm-colab/cache/ddsm_eval.tfrecords" \
    --model_dir="gs://cbis-ddsm-model/`date +%s`" \
    --category_count=4 \
    --training_steps=1000
```

As you can see, the only Cloud ML Engine parameter we changed is `scale-tier`, from `BASIC_GPU` to `BASIC_TPU`. We changed a few custom parameters because the format of the input data is changed for TPU.

Just like GPU training, you can also check the status with the same command (but substitute the job name). Check the status of both training jobs, wait until they finish and check the logs for evaluation results at the end.

Normally TPU training should be much faster than GPU training, however, Cloud ML Engine may spend more time setting up TPUs, so the total time taken on both trainings may look about the same. If you take a look at the logs, it is clear that global steps per second for TPU is 2-3x faster than GPU.

## Model Tuning

Depending on what the results look like, you might want to tweak the training parameters to improve the models. To do this, check out the parameters defined in both gpu_model.py and tpu_model.py, and submit new jobs with new parameters and see whether the results are improved. Note that some advanced parameters may require inline modification.

One good read for this section is the [notes from the Stanford CS class CS231n](http://cs231n.github.io/neural-networks-3/).

This concludes our tutorial and have fun!
