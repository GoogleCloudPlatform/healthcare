# Cloud TPU Tutorial

A TPU named "datathon" is already created in the shared project "".

## Training

Now let's open up a [Cloud Shell](https://console.cloud.google.com/home/dashboard?cloudshell=true) from a browser and run the training task there.

```shell
# "datathon" is the name of the master node.
gcloud compute ssh datathon
```

Next step is downloading the models from Google Cloud Storage.

```shell
gsutil cp -r gs://datathon-cbis-ddsm-colab/cbis_ddsm_ml .
```

Now that we have the model, let's train it on the TPU. Run the following command from the console from previous step.

```shell
python tpu_model.py \
    --tpu=datathon \
    --image_width=95 \
    --image_height=128 \
    --training_data="gs://datathon-cbis-ddsm-colab/cache/ddsm_train.tfrecords" \
    --eval_data="gs://datathon-cbis-ddsm-colab/cache/ddsm_eval.tfrecords" \
    --model_dir="gs://nus-datathon-2018-team-00-shared-files/`date +%s`" \
    --category_count=4 \
    --training_steps=1000
```

To figure out the meaning of these parameters, please run `python tpu_model.py --help`

The whole training process takes about 2-3 minutes. The evaluation result will be printed at the end of the logs.

## Model Tuning

Depending on what the results look like, you might want to tweak the training parameters to improve the models. To do this, check out the parameters defined in tpu_model.py, and run the command above with new parameters and see whether the results are improved. Note that some advanced parameters may require inline modification.

One good read for this section is the [notes from the Stanford CS class CS231n](http://cs231n.github.io/neural-networks-3/).

This concludes our tutorial and have fun!