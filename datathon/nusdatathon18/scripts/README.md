# Scripts for NUS-MIT Datathon

In this folder you can find scripts for preprocessing the CBIS-DDSM dataset, as well as example models for classifying CBIS-DDSM images based on breast density categories.

## Preprocess

It is assumed that the inputs are DICOM files which can be downloaded directly from [cancer imaging archive](https://wiki.cancerimagingarchive.net/display/Public/CBIS-DDSM) (a copy is stored in [cbis-ddsm-images](http://storage.cloud.google.com/cbis-ddsm-images) bucket in GCS), and the output are images in PNG format with dimension specified as parameters. Note that normally you shouldn't need to do this yourself as we already preprocessed all data and the result images are store in [cbis-ddsm-colab](http://storage.cloud.google.com/cbis-ddsm-colab) bucket in GCS.

Please follow the following steps to preprocess the images (these steps were used to generate the preprocessed images). **Make sure to update the flags to match your case, e.g. choose a different `dst_bucket` that you have write access (`cbis-ddsm-colab` is read-only).**

Install all the dependencies required to run the scripts:

```shell
pip install -r requirements.txt
```

Then extract images from DICOM files, the images are stored as TIFF images since they cannot be written as PNGs directly. Note that the maximum dimensions of images will be printed as well, they will be used at later steps.

```shell
# Training data.
python convert_to_tiff.py \
    --src_bucket cbis-ddsm-images \
    --dst_bucket cbis-ddsm-colab \
    --dst_folder train \
    --label_file gs://cbis-ddsm-images/calc_case_description_train_set.csv

# Evaluation data.
python convert_to_tiff.py \
    --src_bucket cbis-ddsm-images \
    --dst_bucket cbis-ddsm-colab \
    --dst_folder test \
    --label_file gs://cbis-ddsm-images/calc_case_description_test_set.csv
```

Next pad and resize the images. Here the `target_with` and `target_height` should be the dimensions from previous step, i.e. the maximum width and height of all images in the dataset. Note that the final sizes of training and test images should be exactly the same.

```shell
# Training data.
python pad_and_resize_images.py \
    --target_width 5251 \
    --target_height 7111 \
    --final_width 95 \
    --final_height 128 \
    --src_bucket cbis-ddsm-colab \
    --dst_bucket cbis-ddsm-colab \
    --dst_folder small_train \
    --src_folder train

# Evaluation data.
python pad_and_resize_images.py \
    --target_width 5251 \
    --target_height 7111 \
    --final_width 95 \
    --final_height 128 \
    --src_bucket cbis-ddsm-colab \
    --dst_bucket cbis-ddsm-colab \
    --dst_folder small_test \
    --src_folder test
```

Now we need to select subset of images for demo purposes. Here we select first 25 training images from each breast density category, and 20 evaluation images randomly.

```shell
python select_demo_images.py \
    --src_bucket cbis-ddsm-colab \
    --src_training_folder small_train \
    --src_eval_folder small_test \
    --dst_bucket cbis-ddsm-colab \
    --dst_training_folder small_train_demo \
    --dst_eval_folder small_test_demo \
    --training_size_per_cat 25 \
    --eval_size 20
```

Finally let's transform the images into a format ([TFRecord](https://www.tensorflow.org/programmers_guide/datasets#consuming_tfrecord_data)) for machine learning models.

```shell
# Training data.
python build_tf_record_dataset.py \
    --src_bucket cbis-ddsm-colab \
    --src_folder small_train \
    --dst_bucket cbis-ddsm-colab \
    --dst_file cache/ddsm_train.tfrecords

# Evaluation data.
python build_tf_record_dataset.py \
    --src_bucket cbis-ddsm-colab \
    --src_folder small_test \
    --dst_bucket cbis-ddsm-colab \
    --dst_file cache/ddsm_eval.tfrecords
```

## ML Models

Please check out the [tutorials folder](../tutorials) for tuturials on using these models.