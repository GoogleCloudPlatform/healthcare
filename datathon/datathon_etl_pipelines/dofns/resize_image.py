"""An Apache Beam DoFn for resizing images."""
import apache_beam as beam
import tensorflow as tf


class ResizeImage(beam.DoFn):
  """Resize an image that is represented as an encoded byte-string.

  Supports PNG and JPEG images.

  Args:
    height (int): the height to resize the images to.
    width (int): the width to resize the images to.
    color_channels (int, default=None): the number of colour channels to use in
      the output image. Defaults to the same number as the input.
    image_format (Union['jpg', 'png'], default='png'): the format of input and
      output images.
  """

  def __init__(self, image_format, height, width, color_channels=None):
    super(ResizeImage, self).__init__()
    if image_format == 'jpeg':
      image_format = 'jpg'
    if image_format not in ('jpg', 'png'):
      raise ValueError('Unrecognized image format ' + image_format)
    self.image_format = image_format
    self.image_shape = (height, width)
    if color_channels is None:
      # Match the color channels of the input image using tf.image.decode_image
      self.image_channels = 0
    else:
      self.image_channels = color_channels

    self.initialized = False
    # Not serializable, to be initialized on each worker
    self._input_bytes_tensor = None
    self._output_bytes_tensor = None
    self._session = None

  def initialize(self):
    """Initialize the tensorflow graph and session for this worker."""
    self._input_bytes_tensor = tf.placeholder(tf.string, [])

    if self.image_format == 'jpg':
      decode_fn = tf.image.decode_jpeg
      encode_fn = tf.image.encode_jpeg
    elif self.image_format == 'png':
      decode_fn = tf.image.decode_png
      encode_fn = tf.image.encode_png
    else:
      raise ValueError('Unrecognized image format ' + self.image_format)

    u8image = decode_fn(self._input_bytes_tensor, channels=self.image_channels)
    resized_u8image = tf.cast(
        tf.image.resize_images(u8image, self.image_shape), tf.uint8)
    self._output_bytes_tensor = encode_fn(resized_u8image)

    self._session = tf.Session()
    self.initialized = True

  def process(self, element):
    """Overrides beam.DoFn.process.

    Args:
      element (Tuple[Any, bytes]): A key with image bytes. The key is not
        modified.

    Yields:
      np.array: a HWC array of the resized image.
    """
    key, image_bytes = element
    if not self.initialized:
      # Initialize non-serializable data once on each worker
      self.initialize()
    image_bytes = self._session.run(self._output_bytes_tensor,
                                    {self._input_bytes_tensor: image_bytes})
    yield key, image_bytes
