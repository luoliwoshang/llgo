package torch

import (
	_ "unsafe"

	"github.com/goplus/llgo/py"
)

const LLGoPackage = "py.torch"

// SymInt-aware utility for logical negation.
//
//	Args:
//	    a (SymBool or bool): Object to negate
//
//go:linkname SymNot py.sym_not
func SymNot(a *py.Object) *py.Object

// SymInt-aware utility for float casting.
//
//	Args:
//	    a (SymInt, SymFloat, or object): Object to cast
//
//go:linkname SymFloat py.sym_float
func SymFloat(a *py.Object) *py.Object

// SymInt-aware utility for int casting.
//
//	Args:
//	    a (SymInt, SymFloat, or object): Object to cast
//
//go:linkname SymInt py.sym_int
func SymInt(a *py.Object) *py.Object

// SymInt-aware utility for max().
//
//go:linkname SymMax py.sym_max
func SymMax(a *py.Object, b *py.Object) *py.Object

// SymInt-aware utility for max().
//
//go:linkname SymMin py.sym_min
func SymMin(a *py.Object, b *py.Object) *py.Object

// Returns True if `obj` is a PyTorch tensor.
//
//	Note that this function is simply doing ``isinstance(obj, Tensor)``.
//	Using that ``isinstance`` check is better for typechecking with mypy,
//	and more explicit - so it's recommended to use that instead of
//	``is_tensor``.
//
//	Args:
//	    obj (Object): Object to test
//	Example::
//
//	    >>> x = torch.tensor([1, 2, 3])
//	    >>> torch.is_tensor(x)
//	    True
//
//go:linkname IsTensor py.is_tensor
func IsTensor(obj *py.Object) *py.Object

// Returns True if `obj` is a PyTorch storage object.
//
//	Args:
//	    obj (Object): Object to test
//
//go:linkname IsStorage py.is_storage
func IsStorage(obj *py.Object) *py.Object

// Sets the default “torch.Tensor“ to be allocated on “device“.  This
//
//	does not affect factory function calls which are called with an explicit
//	``device`` argument.  Factory calls will be performed as if they
//	were passed ``device`` as an argument.
//
//	To only temporarily change the default device instead of setting it
//	globally, use ``with torch.device(device):`` instead.
//
//	The default device is initially ``cpu``.  If you set the default tensor
//	device to another device (e.g., ``cuda``) without a device index, tensors
//	will be allocated on whatever the current device for the device type,
//	even after :func:`torch.cuda.set_device` is called.
//
//	.. warning::
//
//	    This function imposes a slight performance cost on every Python
//	    call to the torch API (not just factory functions).  If this
//	    is causing problems for you, please comment on
//	    https://github.com/pytorch/pytorch/issues/92701
//
//	.. note::
//
//	    This doesn't affect functions that create tensors that share the same memory as the input, like:
//	    :func:`torch.from_numpy` and :func:`torch.frombuffer`
//
//	Args:
//	    device (device or string): the device to set as default
//
//	Example::
//
//	    >>> # xdoctest: +SKIP("requires cuda, changes global state")
//	    >>> torch.tensor([1.2, 3]).device
//	    device(type='cpu')
//	    >>> torch.set_default_device('cuda')  # current device is 0
//	    >>> torch.tensor([1.2, 3]).device
//	    device(type='cuda', index=0)
//	    >>> torch.set_default_device('cuda:1')
//	    >>> torch.tensor([1.2, 3]).device
//	    device(type='cuda', index=1)
//
//go:linkname SetDefaultDevice py.set_default_device
func SetDefaultDevice(device *py.Object) *py.Object

// Sets the default “torch.Tensor“ type to floating point tensor type
//
//	``t``. This type will also be used as default floating point type for
//	type inference in :func:`torch.tensor`.
//
//	The default floating point tensor type is initially ``torch.FloatTensor``.
//
//	Args:
//	    t (type or string): the floating point tensor type or its name
//
//	Example::
//
//	    >>> # xdoctest: +SKIP("Other tests may have changed the default type. Can we reset it?")
//	    >>> torch.tensor([1.2, 3]).dtype    # initial default for floating point is torch.float32
//	    torch.float32
//	    >>> torch.set_default_tensor_type(torch.DoubleTensor)
//	    >>> torch.tensor([1.2, 3]).dtype    # a new floating point tensor
//	    torch.float64
//
//go:linkname SetDefaultTensorType py.set_default_tensor_type
func SetDefaultTensorType(t *py.Object) *py.Object

// Sets the default floating point dtype to :attr:`d`. Supports torch.float32
// and torch.float64 as inputs. Other dtypes may be accepted without complaint
// but are not supported and are unlikely to work as expected.
//
// When PyTorch is initialized its default floating point dtype is torch.float32,
// and the intent of set_default_dtype(torch.float64) is to facilitate NumPy-like
// type inference. The default floating point dtype is used to:
//
//  1. Implicitly determine the default complex dtype. When the default floating point
//     type is float32 the default complex dtype is complex64, and when the default
//     floating point type is float64 the default complex type is complex128.
//  2. Infer the dtype for tensors constructed using Python floats or complex Python
//     numbers. See examples below.
//  3. Determine the result of type promotion between bool and integer tensors and
//     Python floats and complex Python numbers.
//
// Args:
//
//	d (:class:`torch.dtype`): the floating point dtype to make the default.
//	                          Either torch.float32 or torch.float64.
//
// Example:
//
//	>>> # xdoctest: +SKIP("Other tests may have changed the default type. Can we reset it?")
//	>>> # initial default for floating point is torch.float32
//	>>> # Python floats are interpreted as float32
//	>>> torch.tensor([1.2, 3]).dtype
//	torch.float32
//	>>> # initial default for floating point is torch.complex64
//	>>> # Complex Python numbers are interpreted as complex64
//	>>> torch.tensor([1.2, 3j]).dtype
//	torch.complex64
//
//	>>> torch.set_default_dtype(torch.float64)
//
//	>>> # Python floats are now interpreted as float64
//	>>> torch.tensor([1.2, 3]).dtype    # a new floating point tensor
//	torch.float64
//	>>> # Complex Python numbers are now interpreted as complex128
//	>>> torch.tensor([1.2, 3j]).dtype   # a new complex tensor
//	torch.complex128
//
//go:linkname SetDefaultDtype py.set_default_dtype
func SetDefaultDtype(d *py.Object) *py.Object

// Sets whether PyTorch operations must use "deterministic"
//
//	algorithms. That is, algorithms which, given the same input, and when
//	run on the same software and hardware, always produce the same output.
//	When enabled, operations will use deterministic algorithms when available,
//	and if only nondeterministic algorithms are available they will throw a
//	:class:`RuntimeError` when called.
//
//	.. note:: This setting alone is not always enough to make an application
//	    reproducible. Refer to :ref:`reproducibility` for more information.
//
//	.. note:: :func:`torch.set_deterministic_debug_mode` offers an alternative
//	    interface for this feature.
//
//	The following normally-nondeterministic operations will act
//	deterministically when ``mode=True``:
//
//	    * :class:`torch.nn.Conv1d` when called on CUDA tensor
//	    * :class:`torch.nn.Conv2d` when called on CUDA tensor
//	    * :class:`torch.nn.Conv3d` when called on CUDA tensor
//	    * :class:`torch.nn.ConvTranspose1d` when called on CUDA tensor
//	    * :class:`torch.nn.ConvTranspose2d` when called on CUDA tensor
//	    * :class:`torch.nn.ConvTranspose3d` when called on CUDA tensor
//	    * :class:`torch.nn.ReplicationPad2d` when attempting to differentiate a CUDA tensor
//	    * :func:`torch.bmm` when called on sparse-dense CUDA tensors
//	    * :func:`torch.Tensor.__getitem__` when attempting to differentiate a CPU tensor
//	      and the index is a list of tensors
//	    * :func:`torch.Tensor.index_put` with ``accumulate=False``
//	    * :func:`torch.Tensor.index_put` with ``accumulate=True`` when called on a CPU
//	      tensor
//	    * :func:`torch.Tensor.put_` with ``accumulate=True`` when called on a CPU
//	      tensor
//	    * :func:`torch.Tensor.scatter_add_` when called on a CUDA tensor
//	    * :func:`torch.gather` when called on a CUDA tensor that requires grad
//	    * :func:`torch.index_add` when called on CUDA tensor
//	    * :func:`torch.index_select` when attempting to differentiate a CUDA tensor
//	    * :func:`torch.repeat_interleave` when attempting to differentiate a CUDA tensor
//	    * :func:`torch.Tensor.index_copy` when called on a CPU or CUDA tensor
//	    * :func:`torch.Tensor.scatter` when `src` type is Tensor and called on CUDA tensor
//	    * :func:`torch.Tensor.scatter_reduce` when ``reduce='sum'`` or ``reduce='mean'`` and called on CUDA tensor
//
//	The following normally-nondeterministic operations will throw a
//	:class:`RuntimeError` when ``mode=True``:
//
//	    * :class:`torch.nn.AvgPool3d` when attempting to differentiate a CUDA tensor
//	    * :class:`torch.nn.AdaptiveAvgPool2d` when attempting to differentiate a CUDA tensor
//	    * :class:`torch.nn.AdaptiveAvgPool3d` when attempting to differentiate a CUDA tensor
//	    * :class:`torch.nn.MaxPool3d` when attempting to differentiate a CUDA tensor
//	    * :class:`torch.nn.AdaptiveMaxPool2d` when attempting to differentiate a CUDA tensor
//	    * :class:`torch.nn.FractionalMaxPool2d` when attempting to differentiate a CUDA tensor
//	    * :class:`torch.nn.FractionalMaxPool3d` when attempting to differentiate a CUDA tensor
//	    * :class:`torch.nn.MaxUnpool1d`
//	    * :class:`torch.nn.MaxUnpool2d`
//	    * :class:`torch.nn.MaxUnpool3d`
//	    * :func:`torch.nn.functional.interpolate` when attempting to differentiate a CUDA tensor
//	      and one of the following modes is used:
//
//	      - ``linear``
//	      - ``bilinear``
//	      - ``bicubic``
//	      - ``trilinear``
//
//	    * :class:`torch.nn.ReflectionPad1d` when attempting to differentiate a CUDA tensor
//	    * :class:`torch.nn.ReflectionPad2d` when attempting to differentiate a CUDA tensor
//	    * :class:`torch.nn.ReflectionPad3d` when attempting to differentiate a CUDA tensor
//	    * :class:`torch.nn.ReplicationPad1d` when attempting to differentiate a CUDA tensor
//	    * :class:`torch.nn.ReplicationPad3d` when attempting to differentiate a CUDA tensor
//	    * :class:`torch.nn.NLLLoss` when called on a CUDA tensor
//	    * :class:`torch.nn.CTCLoss` when attempting to differentiate a CUDA tensor
//	    * :class:`torch.nn.EmbeddingBag` when attempting to differentiate a CUDA tensor when
//	      ``mode='max'``
//	    * :func:`torch.Tensor.put_` when ``accumulate=False``
//	    * :func:`torch.Tensor.put_` when ``accumulate=True`` and called on a CUDA tensor
//	    * :func:`torch.histc` when called on a CUDA tensor
//	    * :func:`torch.bincount` when called on a CUDA tensor and ``weights``
//	      tensor is given
//	    * :func:`torch.kthvalue` with called on a CUDA tensor
//	    * :func:`torch.median` with indices output when called on a CUDA tensor
//	    * :func:`torch.nn.functional.grid_sample` when attempting to differentiate a CUDA tensor
//	    * :func:`torch.cumsum` when called on a CUDA tensor when dtype is floating point or complex
//	    * :func:`torch.Tensor.scatter_reduce` when ``reduce='prod'`` and called on CUDA tensor
//	    * :func:`torch.Tensor.resize_` when called with a quantized tensor
//
//	In addition, several operations fill uninitialized memory when this setting
//	is turned on and when
//	:attr:`torch.utils.deterministic.fill_uninitialized_memory` is turned on.
//	See the documentation for that attribute for more information.
//
//	A handful of CUDA operations are nondeterministic if the CUDA version is
//	10.2 or greater, unless the environment variable ``CUBLAS_WORKSPACE_CONFIG=:4096:8``
//	or ``CUBLAS_WORKSPACE_CONFIG=:16:8`` is set. See the CUDA documentation for more
//	details: `<https://docs.nvidia.com/cuda/cublas/index.html#cublasApi_reproducibility>`_
//	If one of these environment variable configurations is not set, a :class:`RuntimeError`
//	will be raised from these operations when called with CUDA tensors:
//
//	    * :func:`torch.mm`
//	    * :func:`torch.mv`
//	    * :func:`torch.bmm`
//
//	Note that deterministic operations tend to have worse performance than
//	nondeterministic operations.
//
//	.. note::
//
//	    This flag does not detect or prevent nondeterministic behavior caused
//	    by calling an inplace operation on a tensor with an internal memory
//	    overlap or by giving such a tensor as the :attr:`out` argument for an
//	    operation. In these cases, multiple writes of different data may target
//	    a single memory location, and the order of writes is not guaranteed.
//
//	Args:
//	    mode (:class:`bool`): If True, makes potentially nondeterministic
//	        operations switch to a deterministic algorithm or throw a runtime
//	        error. If False, allows nondeterministic operations.
//
//	Keyword args:
//	    warn_only (:class:`bool`, optional): If True, operations that do not
//	        have a deterministic implementation will throw a warning instead of
//	        an error. Default: ``False``
//
//	Example::
//
//	    >>> # xdoctest: +SKIP
//	    >>> torch.use_deterministic_algorithms(True)
//
//	    # Forward mode nondeterministic error
//	    >>> torch.randn(10, device='cuda').kthvalue(1)
//	    ...
//	    RuntimeError: kthvalue CUDA does not have a deterministic implementation...
//
//	    # Backward mode nondeterministic error
//	    >>> torch.nn.AvgPool3d(1)(torch.randn(3, 4, 5, 6, requires_grad=True).cuda()).sum().backward()
//	    ...
//	    RuntimeError: avg_pool3d_backward_cuda does not have a deterministic implementation...
//
//go:linkname UseDeterministicAlgorithms py.use_deterministic_algorithms
func UseDeterministicAlgorithms(mode *py.Object) *py.Object

// Returns True if the global deterministic flag is turned on. Refer to
//
//	:func:`torch.use_deterministic_algorithms` documentation for more details.
//
//go:linkname AreDeterministicAlgorithmsEnabled py.are_deterministic_algorithms_enabled
func AreDeterministicAlgorithmsEnabled() *py.Object

// Returns True if the global deterministic flag is set to warn only.
//
//	Refer to :func:`torch.use_deterministic_algorithms` documentation for more
//	details.
//
//go:linkname IsDeterministicAlgorithmsWarnOnlyEnabled py.is_deterministic_algorithms_warn_only_enabled
func IsDeterministicAlgorithmsWarnOnlyEnabled() *py.Object

// Sets the debug mode for deterministic operations.
//
//	.. note:: This is an alternative interface for
//	    :func:`torch.use_deterministic_algorithms`. Refer to that function's
//	    documentation for details about affected operations.
//
//	Args:
//	    debug_mode(str or int): If "default" or 0, don't error or warn on
//	        nondeterministic operations. If "warn" or 1, warn on
//	        nondeterministic operations. If "error" or 2, error on
//	        nondeterministic operations.
//
//go:linkname SetDeterministicDebugMode py.set_deterministic_debug_mode
func SetDeterministicDebugMode(debugMode *py.Object) *py.Object

// Returns the current value of the debug mode for deterministic
//
//	operations. Refer to :func:`torch.set_deterministic_debug_mode`
//	documentation for more details.
//
//go:linkname GetDeterministicDebugMode py.get_deterministic_debug_mode
func GetDeterministicDebugMode() *py.Object

// Returns the current value of float32 matrix multiplication precision. Refer to
//
//	:func:`torch.set_float32_matmul_precision` documentation for more details.
//
//go:linkname GetFloat32MatmulPrecision py.get_float32_matmul_precision
func GetFloat32MatmulPrecision() *py.Object

// Sets the internal precision of float32 matrix multiplications.
//
//	Running float32 matrix multiplications in lower precision may significantly increase
//	performance, and in some programs the loss of precision has a negligible impact.
//
//	Supports three settings:
//
//	    * "highest", float32 matrix multiplications use the float32 datatype (24 mantissa
//	      bits) for internal computations.
//	    * "high", float32 matrix multiplications either use the TensorFloat32 datatype (10
//	      mantissa bits) or treat each float32 number as the sum of two bfloat16 numbers
//	      (approximately 16 mantissa bits), if the appropriate fast matrix multiplication
//	      algorithms are available.  Otherwise float32 matrix multiplications are computed
//	      as if the precision is "highest".  See below for more information on the bfloat16
//	      approach.
//	    * "medium", float32 matrix multiplications use the bfloat16 datatype (8 mantissa
//	      bits) for internal computations, if a fast matrix multiplication algorithm
//	      using that datatype internally is available. Otherwise float32
//	      matrix multiplications are computed as if the precision is "high".
//
//	When using "high" precision, float32 multiplications may use a bfloat16-based algorithm
//	that is more complicated than simply truncating to some smaller number mantissa bits
//	(e.g. 10 for TensorFloat32, 8 for bfloat16).  Refer to [Henry2019]_ for a complete
//	description of this algorithm.  To briefly explain here, the first step is to realize
//	that we can perfectly encode a single float32 number as the sum of three bfloat16
//	numbers (because float32 has 24 mantissa bits while bfloat16 has 8, and both have the
//	same number of exponent bits).  This means that the product of two float32 numbers can
//	be exactly given by the sum of nine products of bfloat16 numbers.  We can then trade
//	accuracy for speed by dropping some of these products.  The "high" precision algorithm
//	specifically keeps only the three most significant products, which conveniently excludes
//	all of the products involving the last 8 mantissa bits of either input.  This means that
//	we can represent our inputs as the sum of two bfloat16 numbers rather than three.
//	Because bfloat16 fused-multiply-add (FMA) instructions are typically >10x faster than
//	float32 ones, it's faster to do three multiplications and 2 additions with bfloat16
//	precision than it is to do a single multiplication with float32 precision.
//
//	.. [Henry2019] http://arxiv.org/abs/1904.06376
//
//	.. note::
//
//	    This does not change the output dtype of float32 matrix multiplications,
//	    it controls how the internal computation of the matrix multiplication is performed.
//
//	.. note::
//
//	    This does not change the precision of convolution operations. Other flags,
//	    like `torch.backends.cudnn.allow_tf32`, may control the precision of convolution
//	    operations.
//
//	.. note::
//
//	    This flag currently only affects one native device type: CUDA.
//	    If "high" or "medium" are set then the TensorFloat32 datatype will be used
//	    when computing float32 matrix multiplications, equivalent to setting
//	    `torch.backends.cuda.matmul.allow_tf32 = True`. When "highest" (the default)
//	    is set then the float32 datatype is used for internal computations, equivalent
//	    to setting `torch.backends.cuda.matmul.allow_tf32 = False`.
//
//	Args:
//	    precision(str): can be set to "highest" (default), "high", or "medium" (see above).
//
//go:linkname SetFloat32MatmulPrecision py.set_float32_matmul_precision
func SetFloat32MatmulPrecision(precision *py.Object) *py.Object

// When this flag is False (default) then some PyTorch warnings may only
//
//	appear once per process. This helps avoid excessive warning information.
//	Setting it to True causes these warnings to always appear, which may be
//	helpful when debugging.
//
//	Args:
//	    b (:class:`bool`): If True, force warnings to always be emitted
//	                       If False, set to the default behaviour
//
//go:linkname SetWarnAlways py.set_warn_always
func SetWarnAlways(b *py.Object) *py.Object

// Returns True if the global warn_always flag is turned on. Refer to
//
//	:func:`torch.set_warn_always` documentation for more details.
//
//go:linkname IsWarnAlwaysEnabled py.is_warn_always_enabled
func IsWarnAlwaysEnabled() *py.Object

// Sets the random number generator state.
//
//	.. note: This function only works for CPU. For CUDA, please use
//	         torch.manual_seed(seed), which works for both CPU and CUDA.
//
//	Args:
//	    new_state (torch.ByteTensor): The desired state
//
//go:linkname SetRngState py.set_rng_state
func SetRngState(newState *py.Object) *py.Object

// Returns the random number generator state as a `torch.ByteTensor`.
//
//go:linkname GetRngState py.get_rng_state
func GetRngState() *py.Object

// Sets the seed for generating random numbers. Returns a
//
//	`torch.Generator` object.
//
//	Args:
//	    seed (int): The desired seed. Value must be within the inclusive range
//	        `[-0x8000_0000_0000_0000, 0xffff_ffff_ffff_ffff]`. Otherwise, a RuntimeError
//	        is raised. Negative inputs are remapped to positive values with the formula
//	        `0xffff_ffff_ffff_ffff + seed`.
//
//go:linkname ManualSeed py.manual_seed
func ManualSeed(seed *py.Object) *py.Object

// Returns the initial seed for generating random numbers as a
//
//	Python `long`.
//
//go:linkname InitialSeed py.initial_seed
func InitialSeed() *py.Object

// Sets the seed for generating random numbers to a non-deterministic
//
//	random number. Returns a 64 bit number used to seed the RNG.
//
//go:linkname Seed py.seed
func Seed() *py.Object

// save(obj, f, pickle_module=pickle, pickle_protocol=DEFAULT_PROTOCOL, _use_new_zipfile_serialization=True)
//
//	Saves an object to a disk file.
//
//	See also: :ref:`saving-loading-tensors`
//
//	Args:
//	    obj: saved object
//	    f: a file-like object (has to implement write and flush) or a string or
//	       os.PathLike object containing a file name
//	    pickle_module: module used for pickling metadata and objects
//	    pickle_protocol: can be specified to override the default protocol
//
//	.. note::
//	    A common PyTorch convention is to save tensors using .pt file extension.
//
//	.. note::
//	    PyTorch preserves storage sharing across serialization. See
//	    :ref:`preserve-storage-sharing` for more details.
//
//	.. note::
//	    The 1.6 release of PyTorch switched ``torch.save`` to use a new
//	    zipfile-based file format. ``torch.load`` still retains the ability to
//	    load files in the old format. If for any reason you want ``torch.save``
//	    to use the old format, pass the kwarg ``_use_new_zipfile_serialization=False``.
//
//	Example:
//	    >>> # xdoctest: +SKIP("makes cwd dirty")
//	    >>> # Save to file
//	    >>> x = torch.tensor([0, 1, 2, 3, 4])
//	    >>> torch.save(x, 'tensor.pt')
//	    >>> # Save to io.BytesIO buffer
//	    >>> buffer = io.BytesIO()
//	    >>> torch.save(x, buffer)
//
//go:linkname Save py.save
func Save(obj *py.Object, f *py.Object, pickleModule *py.Object, pickleProtocol *py.Object, UseNewZipfileSerialization *py.Object, DisableByteorderRecord *py.Object) *py.Object

// load(f, map_location=None, pickle_module=pickle, *, weights_only=False, mmap=None, **pickle_load_args)
//
//	Loads an object saved with :func:`torch.save` from a file.
//
//	:func:`torch.load` uses Python's unpickling facilities but treats storages,
//	which underlie tensors, specially. They are first deserialized on the
//	CPU and are then moved to the device they were saved from. If this fails
//	(e.g. because the run time system doesn't have certain devices), an exception
//	is raised. However, storages can be dynamically remapped to an alternative
//	set of devices using the :attr:`map_location` argument.
//
//	If :attr:`map_location` is a callable, it will be called once for each serialized
//	storage with two arguments: storage and location. The storage argument
//	will be the initial deserialization of the storage, residing on the CPU.
//	Each serialized storage has a location tag associated with it which
//	identifies the device it was saved from, and this tag is the second
//	argument passed to :attr:`map_location`. The builtin location tags are ``'cpu'``
//	for CPU tensors and ``'cuda:device_id'`` (e.g. ``'cuda:2'``) for CUDA tensors.
//	:attr:`map_location` should return either ``None`` or a storage. If
//	:attr:`map_location` returns a storage, it will be used as the final deserialized
//	object, already moved to the right device. Otherwise, :func:`torch.load` will
//	fall back to the default behavior, as if :attr:`map_location` wasn't specified.
//
//	If :attr:`map_location` is a :class:`torch.device` object or a string containing
//	a device tag, it indicates the location where all tensors should be loaded.
//
//	Otherwise, if :attr:`map_location` is a dict, it will be used to remap location tags
//	appearing in the file (keys), to ones that specify where to put the
//	storages (values).
//
//	User extensions can register their own location tags and tagging and
//	deserialization methods using :func:`torch.serialization.register_package`.
//
//	Args:
//	    f: a file-like object (has to implement :meth:`read`, :meth:`readline`, :meth:`tell`, and :meth:`seek`),
//	        or a string or os.PathLike object containing a file name
//	    map_location: a function, :class:`torch.device`, string or a dict specifying how to remap storage
//	        locations
//	    pickle_module: module used for unpickling metadata and objects (has to
//	        match the :attr:`pickle_module` used to serialize file)
//	    weights_only: Indicates whether unpickler should be restricted to
//	        loading only tensors, primitive types and dictionaries
//	    mmap: Indicates whether the file should be mmaped rather than loading all the storages into memory.
//	        Typically, tensor storages in the file will first be moved from disk to CPU memory, after which they
//	        are moved to the location that they were tagged with when saving, or specified by ``map_location``. This
//	        second step is a no-op if the final location is CPU. When the ``mmap`` flag is set, instead of copying the
//	        tensor storages from disk to CPU memory in the first step, ``f`` is mmaped.
//	    pickle_load_args: (Python 3 only) optional keyword arguments passed over to
//	        :func:`pickle_module.load` and :func:`pickle_module.Unpickler`, e.g.,
//	        :attr:`errors=...`.
//
//	.. warning::
//	    :func:`torch.load()` unless `weights_only` parameter is set to `True`,
//	    uses ``pickle`` module implicitly, which is known to be insecure.
//	    It is possible to construct malicious pickle data which will execute arbitrary code
//	    during unpickling. Never load data that could have come from an untrusted
//	    source in an unsafe mode, or that could have been tampered with. **Only load data you trust**.
//
//	.. note::
//	    When you call :func:`torch.load()` on a file which contains GPU tensors, those tensors
//	    will be loaded to GPU by default. You can call ``torch.load(.., map_location='cpu')``
//	    and then :meth:`load_state_dict` to avoid GPU RAM surge when loading a model checkpoint.
//
//	.. note::
//	    By default, we decode byte strings as ``utf-8``.  This is to avoid a common error
//	    case ``UnicodeDecodeError: 'ascii' codec can't decode byte 0x...``
//	    when loading files saved by Python 2 in Python 3.  If this default
//	    is incorrect, you may use an extra :attr:`encoding` keyword argument to specify how
//	    these objects should be loaded, e.g., :attr:`encoding='latin1'` decodes them
//	    to strings using ``latin1`` encoding, and :attr:`encoding='bytes'` keeps them
//	    as byte arrays which can be decoded later with ``byte_array.decode(...)``.
//
//	Example:
//	    >>> # xdoctest: +SKIP("undefined filepaths")
//	    >>> torch.load('tensors.pt', weights_only=True)
//	    # Load all tensors onto the CPU
//	    >>> torch.load('tensors.pt', map_location=torch.device('cpu'), weights_only=True)
//	    # Load all tensors onto the CPU, using a function
//	    >>> torch.load('tensors.pt', map_location=lambda storage, loc: storage, weights_only=True)
//	    # Load all tensors onto GPU 1
//	    >>> torch.load('tensors.pt', map_location=lambda storage, loc: storage.cuda(1), weights_only=True)
//	    # Map tensors from GPU 1 to GPU 0
//	    >>> torch.load('tensors.pt', map_location={'cuda:1': 'cuda:0'}, weights_only=True)
//	    # Load tensor from io.BytesIO object
//	    # Loading from a buffer setting weights_only=False, warning this can be unsafe
//	    >>> with open('tensor.pt', 'rb') as f:
//	    ...     buffer = io.BytesIO(f.read())
//	    >>> torch.load(buffer, weights_only=False)
//	    # Load a module with 'ascii' encoding for unpickling
//	    # Loading from a module setting weights_only=False, warning this can be unsafe
//	    >>> torch.load('module.pt', encoding='ascii', weights_only=False)
//
//go:linkname Load py.load
func Load(f *py.Object, mapLocation *py.Object, pickleModule *py.Object) *py.Object

// Set options for printing. Items shamelessly taken from NumPy
//
//	Args:
//	    precision: Number of digits of precision for floating point output
//	        (default = 4).
//	    threshold: Total number of array elements which trigger summarization
//	        rather than full `repr` (default = 1000).
//	    edgeitems: Number of array items in summary at beginning and end of
//	        each dimension (default = 3).
//	    linewidth: The number of characters per line for the purpose of
//	        inserting line breaks (default = 80). Thresholded matrices will
//	        ignore this parameter.
//	    profile: Sane defaults for pretty printing. Can override with any of
//	        the above options. (any one of `default`, `short`, `full`)
//	    sci_mode: Enable (True) or disable (False) scientific notation. If
//	        None (default) is specified, the value is defined by
//	        `torch._tensor_str._Formatter`. This value is automatically chosen
//	        by the framework.
//
//	Example::
//
//	    >>> # Limit the precision of elements
//	    >>> torch.set_printoptions(precision=2)
//	    >>> torch.tensor([1.12345])
//	    tensor([1.12])
//	    >>> # Limit the number of elements shown
//	    >>> torch.set_printoptions(threshold=5)
//	    >>> torch.arange(10)
//	    tensor([0, 1, 2, ..., 7, 8, 9])
//	    >>> # Restore defaults
//	    >>> torch.set_printoptions(profile='default')
//	    >>> torch.tensor([1.12345])
//	    tensor([1.1235])
//	    >>> torch.arange(10)
//	    tensor([0, 1, 2, 3, 4, 5, 6, 7, 8, 9])
//
//go:linkname SetPrintoptions py.set_printoptions
func SetPrintoptions(precision *py.Object, threshold *py.Object, edgeitems *py.Object, linewidth *py.Object, profile *py.Object, sciMode *py.Object) *py.Object

// Returns a 1-dimensional view of each input tensor with zero dimensions.
// Input tensors with one or more dimensions are returned as-is.
//
// Args:
//
//	input (Tensor or list of Tensors)
//
// Returns:
//
//	output (Tensor or tuple of Tensors)
//
// Example::
//
//	>>> x = torch.arange(2)
//	>>> x
//	tensor([0, 1])
//	>>> torch.atleast_1d(x)
//	tensor([0, 1])
//	>>> x = torch.tensor(1.)
//	>>> x
//	tensor(1.)
//	>>> torch.atleast_1d(x)
//	tensor([1.])
//	>>> x = torch.tensor(0.5)
//	>>> y = torch.tensor(1.)
//	>>> torch.atleast_1d((x, y))
//	(tensor([0.5000]), tensor([1.]))
//
//go:linkname Atleast1d py.atleast_1d
func Atleast1d(__llgo_va_list ...interface{}) *py.Object

// Returns a 2-dimensional view of each input tensor with zero dimensions.
// Input tensors with two or more dimensions are returned as-is.
//
// Args:
//
//	input (Tensor or list of Tensors)
//
// Returns:
//
//	output (Tensor or tuple of Tensors)
//
// Example::
//
//	>>> x = torch.tensor(1.)
//	>>> x
//	tensor(1.)
//	>>> torch.atleast_2d(x)
//	tensor([[1.]])
//	>>> x = torch.arange(4).view(2, 2)
//	>>> x
//	tensor([[0, 1],
//	        [2, 3]])
//	>>> torch.atleast_2d(x)
//	tensor([[0, 1],
//	        [2, 3]])
//	>>> x = torch.tensor(0.5)
//	>>> y = torch.tensor(1.)
//	>>> torch.atleast_2d((x, y))
//	(tensor([[0.5000]]), tensor([[1.]]))
//
//go:linkname Atleast2d py.atleast_2d
func Atleast2d(__llgo_va_list ...interface{}) *py.Object

// Returns a 3-dimensional view of each input tensor with zero dimensions.
// Input tensors with three or more dimensions are returned as-is.
//
// Args:
//
//	input (Tensor or list of Tensors)
//
// Returns:
//
//	output (Tensor or tuple of Tensors)
//
// Example:
//
//	>>> x = torch.tensor(0.5)
//	>>> x
//	tensor(0.5000)
//	>>> torch.atleast_3d(x)
//	tensor([[[0.5000]]])
//	>>> y = torch.arange(4).view(2, 2)
//	>>> y
//	tensor([[0, 1],
//	        [2, 3]])
//	>>> torch.atleast_3d(y)
//	tensor([[[0],
//	         [1]],
//	        <BLANKLINE>
//	        [[2],
//	         [3]]])
//	>>> x = torch.tensor(1).view(1, 1, 1)
//	>>> x
//	tensor([[[1]]])
//	>>> torch.atleast_3d(x)
//	tensor([[[1]]])
//	>>> x = torch.tensor(0.5)
//	>>> y = torch.tensor(1.)
//	>>> torch.atleast_3d((x, y))
//	(tensor([[[0.5000]]]), tensor([[[1.]]]))
//
//go:linkname Atleast3d py.atleast_3d
func Atleast3d(__llgo_va_list ...interface{}) *py.Object

// Create a block diagonal matrix from provided tensors.
//
//	Args:
//	    *tensors: One or more tensors with 0, 1, or 2 dimensions.
//
//	Returns:
//	    Tensor: A 2 dimensional tensor with all the input tensors arranged in
//	    order such that their upper left and lower right corners are
//	    diagonally adjacent. All other elements are set to 0.
//
//	Example::
//
//	    >>> import torch
//	    >>> A = torch.tensor([[0, 1], [1, 0]])
//	    >>> B = torch.tensor([[3, 4, 5], [6, 7, 8]])
//	    >>> C = torch.tensor(7)
//	    >>> D = torch.tensor([1, 2, 3])
//	    >>> E = torch.tensor([[4], [5], [6]])
//	    >>> torch.block_diag(A, B, C, D, E)
//	    tensor([[0, 1, 0, 0, 0, 0, 0, 0, 0, 0],
//	            [1, 0, 0, 0, 0, 0, 0, 0, 0, 0],
//	            [0, 0, 3, 4, 5, 0, 0, 0, 0, 0],
//	            [0, 0, 6, 7, 8, 0, 0, 0, 0, 0],
//	            [0, 0, 0, 0, 0, 7, 0, 0, 0, 0],
//	            [0, 0, 0, 0, 0, 0, 1, 2, 3, 0],
//	            [0, 0, 0, 0, 0, 0, 0, 0, 0, 4],
//	            [0, 0, 0, 0, 0, 0, 0, 0, 0, 5],
//	            [0, 0, 0, 0, 0, 0, 0, 0, 0, 6]])
//
//go:linkname BlockDiag py.block_diag
func BlockDiag(__llgo_va_list ...interface{}) *py.Object

// broadcast_tensors(*tensors) -> List of Tensors
//
//	Broadcasts the given tensors according to :ref:`broadcasting-semantics`.
//
//	Args:
//	    *tensors: any number of tensors of the same type
//
//	.. warning::
//
//	    More than one element of a broadcasted tensor may refer to a single
//	    memory location. As a result, in-place operations (especially ones that
//	    are vectorized) may result in incorrect behavior. If you need to write
//	    to the tensors, please clone them first.
//
//	Example::
//
//	    >>> x = torch.arange(3).view(1, 3)
//	    >>> y = torch.arange(2).view(2, 1)
//	    >>> a, b = torch.broadcast_tensors(x, y)
//	    >>> a.size()
//	    torch.Size([2, 3])
//	    >>> a
//	    tensor([[0, 1, 2],
//	            [0, 1, 2]])
//
//go:linkname BroadcastTensors py.broadcast_tensors
func BroadcastTensors(__llgo_va_list ...interface{}) *py.Object

// Do cartesian product of the given sequence of tensors. The behavior is similar to
//
//	python's `itertools.product`.
//
//	Args:
//	    *tensors: any number of 1 dimensional tensors.
//
//	Returns:
//	    Tensor: A tensor equivalent to converting all the input tensors into lists,
//	    do `itertools.product` on these lists, and finally convert the resulting list
//	    into tensor.
//
//	Example::
//
//	    >>> import itertools
//	    >>> a = [1, 2, 3]
//	    >>> b = [4, 5]
//	    >>> list(itertools.product(a, b))
//	    [(1, 4), (1, 5), (2, 4), (2, 5), (3, 4), (3, 5)]
//	    >>> tensor_a = torch.tensor(a)
//	    >>> tensor_b = torch.tensor(b)
//	    >>> torch.cartesian_prod(tensor_a, tensor_b)
//	    tensor([[1, 4],
//	            [1, 5],
//	            [2, 4],
//	            [2, 5],
//	            [3, 4],
//	            [3, 5]])
//
//go:linkname CartesianProd py.cartesian_prod
func CartesianProd(__llgo_va_list ...interface{}) *py.Object

// Computes batched the p-norm distance between each pair of the two collections of row vectors.
//
//	Args:
//	    x1 (Tensor): input tensor of shape :math:`B \times P \times M`.
//	    x2 (Tensor): input tensor of shape :math:`B \times R \times M`.
//	    p: p value for the p-norm distance to calculate between each vector pair
//	        :math:`\in [0, \infty]`.
//	    compute_mode:
//	        'use_mm_for_euclid_dist_if_necessary' - will use matrix multiplication approach to calculate
//	        euclidean distance (p = 2) if P > 25 or R > 25
//	        'use_mm_for_euclid_dist' - will always use matrix multiplication approach to calculate
//	        euclidean distance (p = 2)
//	        'donot_use_mm_for_euclid_dist' - will never use matrix multiplication approach to calculate
//	        euclidean distance (p = 2)
//	        Default: use_mm_for_euclid_dist_if_necessary.
//
//	If x1 has shape :math:`B \times P \times M` and x2 has shape :math:`B \times R \times M` then the
//	output will have shape :math:`B \times P \times R`.
//
//	This function is equivalent to `scipy.spatial.distance.cdist(input,'minkowski', p=p)`
//	if :math:`p \in (0, \infty)`. When :math:`p = 0` it is equivalent to
//	`scipy.spatial.distance.cdist(input, 'hamming') * M`. When :math:`p = \infty`, the closest
//	scipy function is `scipy.spatial.distance.cdist(xn, lambda x, y: np.abs(x - y).max())`.
//
//	Example:
//
//	    >>> a = torch.tensor([[0.9041,  0.0196], [-0.3108, -2.4423], [-0.4821,  1.059]])
//	    >>> a
//	    tensor([[ 0.9041,  0.0196],
//	            [-0.3108, -2.4423],
//	            [-0.4821,  1.0590]])
//	    >>> b = torch.tensor([[-2.1763, -0.4713], [-0.6986,  1.3702]])
//	    >>> b
//	    tensor([[-2.1763, -0.4713],
//	            [-0.6986,  1.3702]])
//	    >>> torch.cdist(a, b, p=2)
//	    tensor([[3.1193, 2.0959],
//	            [2.7138, 3.8322],
//	            [2.2830, 0.3791]])
//
//go:linkname Cdist py.cdist
func Cdist(x1 *py.Object, x2 *py.Object, p *py.Object, computeMode *py.Object) *py.Object

// Returns the matrix product of the :math:`N` 2-D tensors. This product is efficiently computed
//
//	using the matrix chain order algorithm which selects the order in which incurs the lowest cost in terms
//	of arithmetic operations (`[CLRS]`_). Note that since this is a function to compute the product, :math:`N`
//	needs to be greater than or equal to 2; if equal to 2 then a trivial matrix-matrix product is returned.
//	If :math:`N` is 1, then this is a no-op - the original matrix is returned as is.
//
//	.. warning::
//
//	    :func:`torch.chain_matmul` is deprecated and will be removed in a future PyTorch release.
//	    Use :func:`torch.linalg.multi_dot` instead, which accepts a list of two or more tensors
//	    rather than multiple arguments.
//
//	Args:
//	    matrices (Tensors...): a sequence of 2 or more 2-D tensors whose product is to be determined.
//	    out (Tensor, optional): the output tensor. Ignored if :attr:`out` = ``None``.
//
//	Returns:
//	    Tensor: if the :math:`i^{th}` tensor was of dimensions :math:`p_{i} \times p_{i + 1}`, then the product
//	    would be of dimensions :math:`p_{1} \times p_{N + 1}`.
//
//	Example::
//
//	    >>> # xdoctest: +SKIP
//	    >>> # xdoctest: +IGNORE_WANT("non-deterministic")
//	    >>> a = torch.randn(3, 4)
//	    >>> b = torch.randn(4, 5)
//	    >>> c = torch.randn(5, 6)
//	    >>> d = torch.randn(6, 7)
//	    >>> # will raise a deprecation warning
//	    >>> torch.chain_matmul(a, b, c, d)
//	    tensor([[ -2.3375,  -3.9790,  -4.1119,  -6.6577,   9.5609, -11.5095,  -3.2614],
//	            [ 21.4038,   3.3378,  -8.4982,  -5.2457, -10.2561,  -2.4684,   2.7163],
//	            [ -0.9647,  -5.8917,  -2.3213,  -5.2284,  12.8615, -12.2816,  -2.5095]])
//
//	.. _`[CLRS]`: https://mitpress.mit.edu/books/introduction-algorithms-third-edition
//
//go:linkname ChainMatmul py.chain_matmul
func ChainMatmul(__llgo_va_list ...interface{}) *py.Object

// einsum(equation, *operands) -> Tensor
//
//	Sums the product of the elements of the input :attr:`operands` along dimensions specified using a notation
//	based on the Einstein summation convention.
//
//	Einsum allows computing many common multi-dimensional linear algebraic array operations by representing them
//	in a short-hand format based on the Einstein summation convention, given by :attr:`equation`. The details of
//	this format are described below, but the general idea is to label every dimension of the input :attr:`operands`
//	with some subscript and define which subscripts are part of the output. The output is then computed by summing
//	the product of the elements of the :attr:`operands` along the dimensions whose subscripts are not part of the
//	output. For example, matrix multiplication can be computed using einsum as `torch.einsum("ij,jk->ik", A, B)`.
//	Here, j is the summation subscript and i and k the output subscripts (see section below for more details on why).
//
//	Equation:
//
//	    The :attr:`equation` string specifies the subscripts (letters in `[a-zA-Z]`) for each dimension of
//	    the input :attr:`operands` in the same order as the dimensions, separating subscripts for each operand by a
//	    comma (','), e.g. `'ij,jk'` specify subscripts for two 2D operands. The dimensions labeled with the same subscript
//	    must be broadcastable, that is, their size must either match or be `1`. The exception is if a subscript is
//	    repeated for the same input operand, in which case the dimensions labeled with this subscript for this operand
//	    must match in size and the operand will be replaced by its diagonal along these dimensions. The subscripts that
//	    appear exactly once in the :attr:`equation` will be part of the output, sorted in increasing alphabetical order.
//	    The output is computed by multiplying the input :attr:`operands` element-wise, with their dimensions aligned based
//	    on the subscripts, and then summing out the dimensions whose subscripts are not part of the output.
//
//	    Optionally, the output subscripts can be explicitly defined by adding an arrow ('->') at the end of the equation
//	    followed by the subscripts for the output. For instance, the following equation computes the transpose of a
//	    matrix multiplication: 'ij,jk->ki'. The output subscripts must appear at least once for some input operand and
//	    at most once for the output.
//
//	    Ellipsis ('...') can be used in place of subscripts to broadcast the dimensions covered by the ellipsis.
//	    Each input operand may contain at most one ellipsis which will cover the dimensions not covered by subscripts,
//	    e.g. for an input operand with 5 dimensions, the ellipsis in the equation `'ab...c'` cover the third and fourth
//	    dimensions. The ellipsis does not need to cover the same number of dimensions across the :attr:`operands` but the
//	    'shape' of the ellipsis (the size of the dimensions covered by them) must broadcast together. If the output is not
//	    explicitly defined with the arrow ('->') notation, the ellipsis will come first in the output (left-most dimensions),
//	    before the subscript labels that appear exactly once for the input operands. e.g. the following equation implements
//	    batch matrix multiplication `'...ij,...jk'`.
//
//	    A few final notes: the equation may contain whitespaces between the different elements (subscripts, ellipsis,
//	    arrow and comma) but something like `'. . .'` is not valid. An empty string `''` is valid for scalar operands.
//
//	.. note::
//
//	    ``torch.einsum`` handles ellipsis ('...') differently from NumPy in that it allows dimensions
//	    covered by the ellipsis to be summed over, that is, ellipsis are not required to be part of the output.
//
//	.. note::
//
//	    This function uses opt_einsum (https://optimized-einsum.readthedocs.io/en/stable/) to speed up computation or to
//	    consume less memory by optimizing contraction order. This optimization occurs when there are at least three
//	    inputs, since the order does not matter otherwise. Note that finding _the_ optimal path is an NP-hard problem,
//	    thus, opt_einsum relies on different heuristics to achieve near-optimal results. If opt_einsum is not available,
//	    the default order is to contract from left to right.
//
//	    To bypass this default behavior, add the following line to disable the usage of opt_einsum and skip path
//	    calculation: `torch.backends.opt_einsum.enabled = False`
//
//	    To specify which strategy you'd like for opt_einsum to compute the contraction path, add the following line:
//	    `torch.backends.opt_einsum.strategy = 'auto'`. The default strategy is 'auto', and we also support 'greedy' and
//	    'optimal'. Disclaimer that the runtime of 'optimal' is factorial in the number of inputs! See more details in
//	    the opt_einsum documentation (https://optimized-einsum.readthedocs.io/en/stable/path_finding.html).
//
//	.. note::
//
//	    As of PyTorch 1.10 :func:`torch.einsum` also supports the sublist format (see examples below). In this format,
//	    subscripts for each operand are specified by sublists, list of integers in the range [0, 52). These sublists
//	    follow their operands, and an extra sublist can appear at the end of the input to specify the output's
//	    subscripts., e.g. `torch.einsum(op1, sublist1, op2, sublist2, ..., [subslist_out])`. Python's `Ellipsis` object
//	    may be provided in a sublist to enable broadcasting as described in the Equation section above.
//
//	Args:
//	    equation (str): The subscripts for the Einstein summation.
//	    operands (List[Tensor]): The tensors to compute the Einstein summation of.
//
//	Examples::
//
//	    >>> # xdoctest: +IGNORE_WANT("non-deterministic")
//	    >>> # trace
//	    >>> torch.einsum('ii', torch.randn(4, 4))
//	    tensor(-1.2104)
//
//	    >>> # xdoctest: +IGNORE_WANT("non-deterministic")
//	    >>> # diagonal
//	    >>> torch.einsum('ii->i', torch.randn(4, 4))
//	    tensor([-0.1034,  0.7952, -0.2433,  0.4545])
//
//	    >>> # xdoctest: +IGNORE_WANT("non-deterministic")
//	    >>> # outer product
//	    >>> x = torch.randn(5)
//	    >>> y = torch.randn(4)
//	    >>> torch.einsum('i,j->ij', x, y)
//	    tensor([[ 0.1156, -0.2897, -0.3918,  0.4963],
//	            [-0.3744,  0.9381,  1.2685, -1.6070],
//	            [ 0.7208, -1.8058, -2.4419,  3.0936],
//	            [ 0.1713, -0.4291, -0.5802,  0.7350],
//	            [ 0.5704, -1.4290, -1.9323,  2.4480]])
//
//	    >>> # xdoctest: +IGNORE_WANT("non-deterministic")
//	    >>> # batch matrix multiplication
//	    >>> As = torch.randn(3, 2, 5)
//	    >>> Bs = torch.randn(3, 5, 4)
//	    >>> torch.einsum('bij,bjk->bik', As, Bs)
//	    tensor([[[-1.0564, -1.5904,  3.2023,  3.1271],
//	            [-1.6706, -0.8097, -0.8025, -2.1183]],
//
//	            [[ 4.2239,  0.3107, -0.5756, -0.2354],
//	            [-1.4558, -0.3460,  1.5087, -0.8530]],
//
//	            [[ 2.8153,  1.8787, -4.3839, -1.2112],
//	            [ 0.3728, -2.1131,  0.0921,  0.8305]]])
//
//	    >>> # xdoctest: +IGNORE_WANT("non-deterministic")
//	    >>> # with sublist format and ellipsis
//	    >>> torch.einsum(As, [..., 0, 1], Bs, [..., 1, 2], [..., 0, 2])
//	    tensor([[[-1.0564, -1.5904,  3.2023,  3.1271],
//	            [-1.6706, -0.8097, -0.8025, -2.1183]],
//
//	            [[ 4.2239,  0.3107, -0.5756, -0.2354],
//	            [-1.4558, -0.3460,  1.5087, -0.8530]],
//
//	            [[ 2.8153,  1.8787, -4.3839, -1.2112],
//	            [ 0.3728, -2.1131,  0.0921,  0.8305]]])
//
//	    >>> # batch permute
//	    >>> A = torch.randn(2, 3, 4, 5)
//	    >>> torch.einsum('...ij->...ji', A).shape
//	    torch.Size([2, 3, 5, 4])
//
//	    >>> # equivalent to torch.nn.functional.bilinear
//	    >>> A = torch.randn(3, 5, 4)
//	    >>> l = torch.randn(2, 5)
//	    >>> r = torch.randn(2, 4)
//	    >>> torch.einsum('bn,anm,bm->ba', l, A, r)
//	    tensor([[-0.3430, -5.2405,  0.4494],
//	            [ 0.3311,  5.5201, -3.0356]])
//
//go:linkname Einsum py.einsum
func Einsum(__llgo_va_list ...interface{}) *py.Object

// Creates grids of coordinates specified by the 1D inputs in `attr`:tensors.
//
//	This is helpful when you want to visualize data over some
//	range of inputs. See below for a plotting example.
//
//	Given :math:`N` 1D tensors :math:`T_0 \ldots T_{N-1}` as
//	inputs with corresponding sizes :math:`S_0 \ldots S_{N-1}`,
//	this creates :math:`N` N-dimensional tensors :math:`G_0 \ldots
//	G_{N-1}`, each with shape :math:`(S_0, ..., S_{N-1})` where
//	the output :math:`G_i` is constructed by expanding :math:`T_i`
//	to the result shape.
//
//	.. note::
//	    0D inputs are treated equivalently to 1D inputs of a
//	    single element.
//
//	.. warning::
//	    `torch.meshgrid(*tensors)` currently has the same behavior
//	    as calling `numpy.meshgrid(*arrays, indexing='ij')`.
//
//	    In the future `torch.meshgrid` will transition to
//	    `indexing='xy'` as the default.
//
//	    https://github.com/pytorch/pytorch/issues/50276 tracks
//	    this issue with the goal of migrating to NumPy's behavior.
//
//	.. seealso::
//
//	    :func:`torch.cartesian_prod` has the same effect but it
//	    collects the data in a tensor of vectors.
//
//	Args:
//	    tensors (list of Tensor): list of scalars or 1 dimensional tensors. Scalars will be
//	        treated as tensors of size :math:`(1,)` automatically
//
//	    indexing: (str, optional): the indexing mode, either "xy"
//	        or "ij", defaults to "ij". See warning for future changes.
//
//	        If "xy" is selected, the first dimension corresponds
//	        to the cardinality of the second input and the second
//	        dimension corresponds to the cardinality of the first
//	        input.
//
//	        If "ij" is selected, the dimensions are in the same
//	        order as the cardinality of the inputs.
//
//	Returns:
//	    seq (sequence of Tensors): If the input has :math:`N`
//	    tensors of size :math:`S_0 \ldots S_{N-1}``, then the
//	    output will also have :math:`N` tensors, where each tensor
//	    is of shape :math:`(S_0, ..., S_{N-1})`.
//
//	Example::
//
//	    >>> x = torch.tensor([1, 2, 3])
//	    >>> y = torch.tensor([4, 5, 6])
//
//	    Observe the element-wise pairings across the grid, (1, 4),
//	    (1, 5), ..., (3, 6). This is the same thing as the
//	    cartesian product.
//	    >>> grid_x, grid_y = torch.meshgrid(x, y, indexing='ij')
//	    >>> grid_x
//	    tensor([[1, 1, 1],
//	            [2, 2, 2],
//	            [3, 3, 3]])
//	    >>> grid_y
//	    tensor([[4, 5, 6],
//	            [4, 5, 6],
//	            [4, 5, 6]])
//
//	    This correspondence can be seen when these grids are
//	    stacked properly.
//	    >>> torch.equal(torch.cat(tuple(torch.dstack([grid_x, grid_y]))),
//	    ...             torch.cartesian_prod(x, y))
//	    True
//
//	    `torch.meshgrid` is commonly used to produce a grid for
//	    plotting.
//	    >>> # xdoctest: +REQUIRES(module:matplotlib)
//	    >>> # xdoctest: +REQUIRES(env:DOCTEST_SHOW)
//	    >>> import matplotlib.pyplot as plt
//	    >>> xs = torch.linspace(-5, 5, steps=100)
//	    >>> ys = torch.linspace(-5, 5, steps=100)
//	    >>> x, y = torch.meshgrid(xs, ys, indexing='xy')
//	    >>> z = torch.sin(torch.sqrt(x * x + y * y))
//	    >>> ax = plt.axes(projection='3d')
//	    >>> ax.plot_surface(x.numpy(), y.numpy(), z.numpy())
//	    >>> plt.show()
//
//	.. image:: ../_static/img/meshgrid.png
//	    :width: 512
//
//go:linkname Meshgrid py.meshgrid
func Meshgrid(__llgo_va_list ...interface{}) *py.Object

// Returns the matrix norm or vector norm of a given tensor.
//
//	.. warning::
//
//	    torch.norm is deprecated and may be removed in a future PyTorch release.
//	    Its documentation and behavior may be incorrect, and it is no longer
//	    actively maintained.
//
//	    Use :func:`torch.linalg.vector_norm` when computing vector norms and
//	    :func:`torch.linalg.matrix_norm` when computing matrix norms.
//	    For a function with a similar behavior as this one see :func:`torch.linalg.norm`.
//	    Note, however, the signature for these functions is slightly different than the
//	    signature for ``torch.norm``.
//
//	Args:
//	    input (Tensor): The input tensor. Its data type must be either a floating
//	        point or complex type. For complex inputs, the norm is calculated using the
//	        absolute value of each element. If the input is complex and neither
//	        :attr:`dtype` nor :attr:`out` is specified, the result's data type will
//	        be the corresponding floating point type (e.g. float if :attr:`input` is
//	        complexfloat).
//
//	    p (int, float, inf, -inf, 'fro', 'nuc', optional): the order of norm. Default: ``'fro'``
//	        The following norms can be calculated:
//
//	        ======  ==============  ==========================
//	        ord     matrix norm     vector norm
//	        ======  ==============  ==========================
//	        'fro'   Frobenius norm  --
//	        'nuc'   nuclear norm    --
//	        Number  --              sum(abs(x)**ord)**(1./ord)
//	        ======  ==============  ==========================
//
//	        The vector norm can be calculated across any number of dimensions.
//	        The corresponding dimensions of :attr:`input` are flattened into
//	        one dimension, and the norm is calculated on the flattened
//	        dimension.
//
//	        Frobenius norm produces the same result as ``p=2`` in all cases
//	        except when :attr:`dim` is a list of three or more dims, in which
//	        case Frobenius norm throws an error.
//
//	        Nuclear norm can only be calculated across exactly two dimensions.
//
//	    dim (int, tuple of ints, list of ints, optional):
//	        Specifies which dimension or dimensions of :attr:`input` to
//	        calculate the norm across. If :attr:`dim` is ``None``, the norm will
//	        be calculated across all dimensions of :attr:`input`. If the norm
//	        type indicated by :attr:`p` does not support the specified number of
//	        dimensions, an error will occur.
//	    keepdim (bool, optional): whether the output tensors have :attr:`dim`
//	        retained or not. Ignored if :attr:`dim` = ``None`` and
//	        :attr:`out` = ``None``. Default: ``False``
//	    out (Tensor, optional): the output tensor. Ignored if
//	        :attr:`dim` = ``None`` and :attr:`out` = ``None``.
//	    dtype (:class:`torch.dtype`, optional): the desired data type of
//	        returned tensor. If specified, the input tensor is casted to
//	        :attr:`dtype` while performing the operation. Default: None.
//
//	.. note::
//	    Even though ``p='fro'`` supports any number of dimensions, the true
//	    mathematical definition of Frobenius norm only applies to tensors with
//	    exactly two dimensions. :func:`torch.linalg.matrix_norm` with ``ord='fro'``
//	    aligns with the mathematical definition, since it can only be applied across
//	    exactly two dimensions.
//
//	Example::
//
//	    >>> import torch
//	    >>> a = torch.arange(9, dtype= torch.float) - 4
//	    >>> b = a.reshape((3, 3))
//	    >>> torch.norm(a)
//	    tensor(7.7460)
//	    >>> torch.norm(b)
//	    tensor(7.7460)
//	    >>> torch.norm(a, float('inf'))
//	    tensor(4.)
//	    >>> torch.norm(b, float('inf'))
//	    tensor(4.)
//	    >>> c = torch.tensor([[ 1, 2, 3], [-1, 1, 4]] , dtype=torch.float)
//	    >>> torch.norm(c, dim=0)
//	    tensor([1.4142, 2.2361, 5.0000])
//	    >>> torch.norm(c, dim=1)
//	    tensor([3.7417, 4.2426])
//	    >>> torch.norm(c, p=1, dim=1)
//	    tensor([6., 6.])
//	    >>> d = torch.arange(8, dtype=torch.float).reshape(2, 2, 2)
//	    >>> torch.norm(d, dim=(1, 2))
//	    tensor([ 3.7417, 11.2250])
//	    >>> torch.norm(d[0, :, :]), torch.norm(d[1, :, :])
//	    (tensor(3.7417), tensor(11.2250))
//
//go:linkname Norm py.norm
func Norm(input *py.Object, p *py.Object, dim *py.Object, keepdim *py.Object, out *py.Object, dtype *py.Object) *py.Object

// Splits the tensor into chunks. Each chunk is a view of the original tensor.
//
//	If :attr:`split_size_or_sections` is an integer type, then :attr:`tensor` will
//	be split into equally sized chunks (if possible). Last chunk will be smaller if
//	the tensor size along the given dimension :attr:`dim` is not divisible by
//	:attr:`split_size`.
//
//	If :attr:`split_size_or_sections` is a list, then :attr:`tensor` will be split
//	into ``len(split_size_or_sections)`` chunks with sizes in :attr:`dim` according
//	to :attr:`split_size_or_sections`.
//
//	Args:
//	    tensor (Tensor): tensor to split.
//	    split_size_or_sections (int) or (list(int)): size of a single chunk or
//	        list of sizes for each chunk
//	    dim (int): dimension along which to split the tensor.
//
//	Example::
//
//	    >>> a = torch.arange(10).reshape(5, 2)
//	    >>> a
//	    tensor([[0, 1],
//	            [2, 3],
//	            [4, 5],
//	            [6, 7],
//	            [8, 9]])
//	    >>> torch.split(a, 2)
//	    (tensor([[0, 1],
//	             [2, 3]]),
//	     tensor([[4, 5],
//	             [6, 7]]),
//	     tensor([[8, 9]]))
//	    >>> torch.split(a, [1, 4])
//	    (tensor([[0, 1]]),
//	     tensor([[2, 3],
//	             [4, 5],
//	             [6, 7],
//	             [8, 9]]))
//
//go:linkname Split py.split
func Split(tensor *py.Object, splitSizeOrSections *py.Object, dim *py.Object) *py.Object

// Short-time Fourier transform (STFT).
//
//	.. warning::
//	    From version 1.8.0, :attr:`return_complex` must always be given
//	    explicitly for real inputs and `return_complex=False` has been
//	    deprecated. Strongly prefer `return_complex=True` as in a future
//	    pytorch release, this function will only return complex tensors.
//
//	    Note that :func:`torch.view_as_real` can be used to recover a real
//	    tensor with an extra last dimension for real and imaginary components.
//
//	.. warning::
//	    From version 2.1, a warning will be provided if a :attr:`window` is
//	    not specified. In a future release, this attribute will be required.
//	    Not providing a window currently defaults to using a rectangular window,
//	    which may result in undesirable artifacts. Consider using tapered windows,
//	    such as :func:`torch.hann_window`.
//
//	The STFT computes the Fourier transform of short overlapping windows of the
//	input. This giving frequency components of the signal as they change over
//	time. The interface of this function is modeled after (but *not* a drop-in
//	replacement for) librosa_ stft function.
//
//	.. _librosa: https://librosa.org/doc/latest/generated/librosa.stft.html
//
//	Ignoring the optional batch dimension, this method computes the following
//	expression:
//
//	.. math::
//	    X[\omega, m] = \sum_{k = 0}^{\text{win\_length-1}}%
//	                        \text{window}[k]\ \text{input}[m \times \text{hop\_length} + k]\ %
//	                        \exp\left(- j \frac{2 \pi \cdot \omega k}{\text{n\_fft}}\right),
//
//	where :math:`m` is the index of the sliding window, and :math:`\omega` is
//	the frequency :math:`0 \leq \omega < \text{n\_fft}` for ``onesided=False``,
//	or :math:`0 \leq \omega < \lfloor \text{n\_fft} / 2 \rfloor + 1` for ``onesided=True``.
//
//	* :attr:`input` must be either a 1-D time sequence or a 2-D batch of time
//	  sequences.
//
//	* If :attr:`hop_length` is ``None`` (default), it is treated as equal to
//	  ``floor(n_fft / 4)``.
//
//	* If :attr:`win_length` is ``None`` (default), it is treated as equal to
//	  :attr:`n_fft`.
//
//	* :attr:`window` can be a 1-D tensor of size :attr:`win_length`, e.g., from
//	  :meth:`torch.hann_window`. If :attr:`window` is ``None`` (default), it is
//	  treated as if having :math:`1` everywhere in the window. If
//	  :math:`\text{win\_length} < \text{n\_fft}`, :attr:`window` will be padded on
//	  both sides to length :attr:`n_fft` before being applied.
//
//	* If :attr:`center` is ``True`` (default), :attr:`input` will be padded on
//	  both sides so that the :math:`t`-th frame is centered at time
//	  :math:`t \times \text{hop\_length}`. Otherwise, the :math:`t`-th frame
//	  begins at time  :math:`t \times \text{hop\_length}`.
//
//	* :attr:`pad_mode` determines the padding method used on :attr:`input` when
//	  :attr:`center` is ``True``. See :meth:`torch.nn.functional.pad` for
//	  all available options. Default is ``"reflect"``.
//
//	* If :attr:`onesided` is ``True`` (default for real input), only values for
//	  :math:`\omega` in :math:`\left[0, 1, 2, \dots, \left\lfloor
//	  \frac{\text{n\_fft}}{2} \right\rfloor + 1\right]` are returned because
//	  the real-to-complex Fourier transform satisfies the conjugate symmetry,
//	  i.e., :math:`X[m, \omega] = X[m, \text{n\_fft} - \omega]^*`.
//	  Note if the input or window tensors are complex, then :attr:`onesided`
//	  output is not possible.
//
//	* If :attr:`normalized` is ``True`` (default is ``False``), the function
//	  returns the normalized STFT results, i.e., multiplied by :math:`(\text{frame\_length})^{-0.5}`.
//
//	* If :attr:`return_complex` is ``True`` (default if input is complex), the
//	  return is a ``input.dim() + 1`` dimensional complex tensor. If ``False``,
//	  the output is a ``input.dim() + 2`` dimensional real tensor where the last
//	  dimension represents the real and imaginary components.
//
//	Returns either a complex tensor of size :math:`(* \times N \times T)` if
//	:attr:`return_complex` is true, or a real tensor of size :math:`(* \times N
//	\times T \times 2)`. Where :math:`*` is the optional batch size of
//	:attr:`input`, :math:`N` is the number of frequencies where STFT is applied
//	and :math:`T` is the total number of frames used.
//
//	.. warning::
//	  This function changed signature at version 0.4.1. Calling with the
//	  previous signature may cause error or return incorrect result.
//
//	Args:
//	    input (Tensor): the input tensor of shape `(B?, L)` where `B?` is an optional
//	        batch dimension
//	    n_fft (int): size of Fourier transform
//	    hop_length (int, optional): the distance between neighboring sliding window
//	        frames. Default: ``None`` (treated as equal to ``floor(n_fft / 4)``)
//	    win_length (int, optional): the size of window frame and STFT filter.
//	        Default: ``None``  (treated as equal to :attr:`n_fft`)
//	    window (Tensor, optional): the optional window function.
//	        Shape must be 1d and `<= n_fft`
//	        Default: ``None`` (treated as window of all :math:`1` s)
//	    center (bool, optional): whether to pad :attr:`input` on both sides so
//	        that the :math:`t`-th frame is centered at time :math:`t \times \text{hop\_length}`.
//	        Default: ``True``
//	    pad_mode (str, optional): controls the padding method used when
//	        :attr:`center` is ``True``. Default: ``"reflect"``
//	    normalized (bool, optional): controls whether to return the normalized STFT results
//	         Default: ``False``
//	    onesided (bool, optional): controls whether to return half of results to
//	        avoid redundancy for real inputs.
//	        Default: ``True`` for real :attr:`input` and :attr:`window`, ``False`` otherwise.
//	    return_complex (bool, optional): whether to return a complex tensor, or
//	        a real tensor with an extra last dimension for the real and
//	        imaginary components.
//
//	        .. versionchanged:: 2.0
//	           ``return_complex`` is now a required argument for real inputs,
//	           as the default is being transitioned to ``True``.
//
//	        .. deprecated:: 2.0
//	           ``return_complex=False`` is deprecated, instead use ``return_complex=True``
//	           Note that calling :func:`torch.view_as_real` on the output will
//	           recover the deprecated output format.
//
//	Returns:
//	    Tensor: A tensor containing the STFT result with shape `(B?, N, T, C?)` where
//	       - `B?` is an optional batch dimnsion from the input
//	       - `N` is the number of frequency samples, `(n_fft // 2) + 1` for
//	         `onesided=True`, or otherwise `n_fft`.
//	       - `T` is the number of frames, `1 + L // hop_length`
//	         for `center=True`, or `1 + (L - n_fft) // hop_length` otherwise.
//	       - `C?` is an optional length-2 dimension of real and imaginary
//	         components, present when `return_complex=False`.
//
//go:linkname Stft py.stft
func Stft(input *py.Object, nFft *py.Object, hopLength *py.Object, winLength *py.Object, window *py.Object, center *py.Object, padMode *py.Object, normalized *py.Object, onesided *py.Object, returnComplex *py.Object) *py.Object

// Returns a contraction of a and b over multiple dimensions.
//
//	:attr:`tensordot` implements a generalized matrix product.
//
//	Args:
//	  a (Tensor): Left tensor to contract
//	  b (Tensor): Right tensor to contract
//	  dims (int or Tuple[List[int], List[int]] or List[List[int]] containing two lists or Tensor): number of dimensions to
//	     contract or explicit lists of dimensions for :attr:`a` and
//	     :attr:`b` respectively
//
//	When called with a non-negative integer argument :attr:`dims` = :math:`d`, and
//	the number of dimensions of :attr:`a` and :attr:`b` is :math:`m` and :math:`n`,
//	respectively, :func:`~torch.tensordot` computes
//
//	.. math::
//	    r_{i_0,...,i_{m-d}, i_d,...,i_n}
//	      = \sum_{k_0,...,k_{d-1}} a_{i_0,...,i_{m-d},k_0,...,k_{d-1}} \times b_{k_0,...,k_{d-1}, i_d,...,i_n}.
//
//	When called with :attr:`dims` of the list form, the given dimensions will be contracted
//	in place of the last :math:`d` of :attr:`a` and the first :math:`d` of :math:`b`. The sizes
//	in these dimensions must match, but :func:`~torch.tensordot` will deal with broadcasted
//	dimensions.
//
//	Examples::
//
//	    >>> a = torch.arange(60.).reshape(3, 4, 5)
//	    >>> b = torch.arange(24.).reshape(4, 3, 2)
//	    >>> torch.tensordot(a, b, dims=([1, 0], [0, 1]))
//	    tensor([[4400., 4730.],
//	            [4532., 4874.],
//	            [4664., 5018.],
//	            [4796., 5162.],
//	            [4928., 5306.]])
//
//	    >>> # xdoctest: +REQUIRES(env:TORCH_DOCTEST_CUDA)
//	    >>> a = torch.randn(3, 4, 5, device='cuda')
//	    >>> b = torch.randn(4, 5, 6, device='cuda')
//	    >>> c = torch.tensordot(a, b, dims=2).cpu()
//	    tensor([[ 8.3504, -2.5436,  6.2922,  2.7556, -1.0732,  3.2741],
//	            [ 3.3161,  0.0704,  5.0187, -0.4079, -4.3126,  4.8744],
//	            [ 0.8223,  3.9445,  3.2168, -0.2400,  3.4117,  1.7780]])
//
//	    >>> a = torch.randn(3, 5, 4, 6)
//	    >>> b = torch.randn(6, 4, 5, 3)
//	    >>> torch.tensordot(a, b, dims=([2, 1, 3], [1, 2, 0]))
//	    tensor([[  7.7193,  -2.4867, -10.3204],
//	            [  1.5513, -14.4737,  -6.5113],
//	            [ -0.2850,   4.2573,  -3.5997]])
//
//go:linkname Tensordot py.tensordot
func Tensordot(a *py.Object, b *py.Object, dims *py.Object, out *py.Object) *py.Object

// Eliminates all but the first element from every consecutive group of equivalent elements.
//
//	.. note:: This function is different from :func:`torch.unique` in the sense that this function
//	    only eliminates consecutive duplicate values. This semantics is similar to `std::unique`
//	    in C++.
//
//	Args:
//	    input (Tensor): the input tensor
//	    return_inverse (bool): Whether to also return the indices for where
//	        elements in the original input ended up in the returned unique list.
//	    return_counts (bool): Whether to also return the counts for each unique
//	        element.
//	    dim (int): the dimension to apply unique. If ``None``, the unique of the
//	        flattened input is returned. default: ``None``
//
//	Returns:
//	    (Tensor, Tensor (optional), Tensor (optional)): A tensor or a tuple of tensors containing
//
//	        - **output** (*Tensor*): the output list of unique scalar elements.
//	        - **inverse_indices** (*Tensor*): (optional) if
//	          :attr:`return_inverse` is True, there will be an additional
//	          returned tensor (same shape as input) representing the indices
//	          for where elements in the original input map to in the output;
//	          otherwise, this function will only return a single tensor.
//	        - **counts** (*Tensor*): (optional) if
//	          :attr:`return_counts` is True, there will be an additional
//	          returned tensor (same shape as output or output.size(dim),
//	          if dim was specified) representing the number of occurrences
//	          for each unique value or tensor.
//
//	Example::
//
//	    >>> x = torch.tensor([1, 1, 2, 2, 3, 1, 1, 2])
//	    >>> output = torch.unique_consecutive(x)
//	    >>> output
//	    tensor([1, 2, 3, 1, 2])
//
//	    >>> output, inverse_indices = torch.unique_consecutive(x, return_inverse=True)
//	    >>> output
//	    tensor([1, 2, 3, 1, 2])
//	    >>> inverse_indices
//	    tensor([0, 0, 1, 1, 2, 3, 3, 4])
//
//	    >>> output, counts = torch.unique_consecutive(x, return_counts=True)
//	    >>> output
//	    tensor([1, 2, 3, 1, 2])
//	    >>> counts
//	    tensor([2, 2, 1, 2, 1])
//
//go:linkname UniqueConsecutive py.unique_consecutive
func UniqueConsecutive(__llgo_va_list ...interface{}) *py.Object

// broadcast_shapes(*shapes) -> Size
//
//	Similar to :func:`broadcast_tensors` but for shapes.
//
//	This is equivalent to
//	``torch.broadcast_tensors(*map(torch.empty, shapes))[0].shape``
//	but avoids the need create to intermediate tensors. This is useful for
//	broadcasting tensors of common batch shape but different rightmost shape,
//	e.g. to broadcast mean vectors with covariance matrices.
//
//	Example::
//
//	    >>> torch.broadcast_shapes((2,), (3, 1), (1, 1, 1))
//	    torch.Size([1, 3, 2])
//
//	Args:
//	    \*shapes (torch.Size): Shapes of tensors.
//
//	Returns:
//	    shape (torch.Size): A shape compatible with all input shapes.
//
//	Raises:
//	    RuntimeError: If shapes are incompatible.
//
//go:linkname BroadcastShapes py.broadcast_shapes
func BroadcastShapes(__llgo_va_list ...interface{}) *py.Object

// Computes the LU factorization of a matrix or batches of matrices
//
//	:attr:`A`. Returns a tuple containing the LU factorization and
//	pivots of :attr:`A`.  Pivoting is done if :attr:`pivot` is set to
//	``True``.
//
//	.. warning::
//
//	    :func:`torch.lu` is deprecated in favor of :func:`torch.linalg.lu_factor`
//	    and :func:`torch.linalg.lu_factor_ex`. :func:`torch.lu` will be removed in a
//	    future PyTorch release.
//	    ``LU, pivots, info = torch.lu(A, compute_pivots)`` should be replaced with
//
//	    .. code:: python
//
//	        LU, pivots = torch.linalg.lu_factor(A, compute_pivots)
//
//	    ``LU, pivots, info = torch.lu(A, compute_pivots, get_infos=True)`` should be replaced with
//
//	    .. code:: python
//
//	        LU, pivots, info = torch.linalg.lu_factor_ex(A, compute_pivots)
//
//	.. note::
//	    * The returned permutation matrix for every matrix in the batch is
//	      represented by a 1-indexed vector of size ``min(A.shape[-2], A.shape[-1])``.
//	      ``pivots[i] == j`` represents that in the ``i``-th step of the algorithm,
//	      the ``i``-th row was permuted with the ``j-1``-th row.
//	    * LU factorization with :attr:`pivot` = ``False`` is not available
//	      for CPU, and attempting to do so will throw an error. However,
//	      LU factorization with :attr:`pivot` = ``False`` is available for
//	      CUDA.
//	    * This function does not check if the factorization was successful
//	      or not if :attr:`get_infos` is ``True`` since the status of the
//	      factorization is present in the third element of the return tuple.
//	    * In the case of batches of square matrices with size less or equal
//	      to 32 on a CUDA device, the LU factorization is repeated for
//	      singular matrices due to the bug in the MAGMA library
//	      (see magma issue 13).
//	    * ``L``, ``U``, and ``P`` can be derived using :func:`torch.lu_unpack`.
//
//	.. warning::
//	    The gradients of this function will only be finite when :attr:`A` is full rank.
//	    This is because the LU decomposition is just differentiable at full rank matrices.
//	    Furthermore, if :attr:`A` is close to not being full rank,
//	    the gradient will be numerically unstable as it depends on the computation of :math:`L^{-1}` and :math:`U^{-1}`.
//
//	Args:
//	    A (Tensor): the tensor to factor of size :math:`(*, m, n)`
//	    pivot (bool, optional): controls whether pivoting is done. Default: ``True``
//	    get_infos (bool, optional): if set to ``True``, returns an info IntTensor.
//	                                Default: ``False``
//	    out (tuple, optional): optional output tuple. If :attr:`get_infos` is ``True``,
//	                           then the elements in the tuple are Tensor, IntTensor,
//	                           and IntTensor. If :attr:`get_infos` is ``False``, then the
//	                           elements in the tuple are Tensor, IntTensor. Default: ``None``
//
//	Returns:
//	    (Tensor, IntTensor, IntTensor (optional)): A tuple of tensors containing
//
//	        - **factorization** (*Tensor*): the factorization of size :math:`(*, m, n)`
//
//	        - **pivots** (*IntTensor*): the pivots of size :math:`(*, \text{min}(m, n))`.
//	          ``pivots`` stores all the intermediate transpositions of rows.
//	          The final permutation ``perm`` could be reconstructed by
//	          applying ``swap(perm[i], perm[pivots[i] - 1])`` for ``i = 0, ..., pivots.size(-1) - 1``,
//	          where ``perm`` is initially the identity permutation of :math:`m` elements
//	          (essentially this is what :func:`torch.lu_unpack` is doing).
//
//	        - **infos** (*IntTensor*, *optional*): if :attr:`get_infos` is ``True``, this is a tensor of
//	          size :math:`(*)` where non-zero values indicate whether factorization for the matrix or
//	          each minibatch has succeeded or failed
//
//	Example::
//
//	    >>> # xdoctest: +REQUIRES(env:TORCH_DOCTEST_LAPACK)
//	    >>> # xdoctest: +IGNORE_WANT("non-deterministic")
//	    >>> A = torch.randn(2, 3, 3)
//	    >>> A_LU, pivots = torch.lu(A)
//	    >>> A_LU
//	    tensor([[[ 1.3506,  2.5558, -0.0816],
//	             [ 0.1684,  1.1551,  0.1940],
//	             [ 0.1193,  0.6189, -0.5497]],
//
//	            [[ 0.4526,  1.2526, -0.3285],
//	             [-0.7988,  0.7175, -0.9701],
//	             [ 0.2634, -0.9255, -0.3459]]])
//	    >>> pivots
//	    tensor([[ 3,  3,  3],
//	            [ 3,  3,  3]], dtype=torch.int32)
//	    >>> A_LU, pivots, info = torch.lu(A, get_infos=True)
//	    >>> if info.nonzero().size(0) == 0:
//	    ...     print('LU factorization succeeded for all samples!')
//	    LU factorization succeeded for all samples!
//
//go:linkname Lu py.lu
func Lu(__llgo_va_list ...interface{}) *py.Object

// Performs linear Principal Component Analysis (PCA) on a low-rank
//
//	matrix, batches of such matrices, or sparse matrix.
//
//	This function returns a namedtuple ``(U, S, V)`` which is the
//	nearly optimal approximation of a singular value decomposition of
//	a centered matrix :math:`A` such that :math:`A = U diag(S) V^T`.
//
//	.. note:: The relation of ``(U, S, V)`` to PCA is as follows:
//
//	            - :math:`A` is a data matrix with ``m`` samples and
//	              ``n`` features
//
//	            - the :math:`V` columns represent the principal directions
//
//	            - :math:`S ** 2 / (m - 1)` contains the eigenvalues of
//	              :math:`A^T A / (m - 1)` which is the covariance of
//	              ``A`` when ``center=True`` is provided.
//
//	            - ``matmul(A, V[:, :k])`` projects data to the first k
//	              principal components
//
//	.. note:: Different from the standard SVD, the size of returned
//	          matrices depend on the specified rank and q
//	          values as follows:
//
//	            - :math:`U` is m x q matrix
//
//	            - :math:`S` is q-vector
//
//	            - :math:`V` is n x q matrix
//
//	.. note:: To obtain repeatable results, reset the seed for the
//	          pseudorandom number generator
//
//	Args:
//
//	    A (Tensor): the input tensor of size :math:`(*, m, n)`
//
//	    q (int, optional): a slightly overestimated rank of
//	                       :math:`A`. By default, ``q = min(6, m,
//	                       n)``.
//
//	    center (bool, optional): if True, center the input tensor,
//	                             otherwise, assume that the input is
//	                             centered.
//
//	    niter (int, optional): the number of subspace iterations to
//	                           conduct; niter must be a nonnegative
//	                           integer, and defaults to 2.
//
//	References::
//
//	    - Nathan Halko, Per-Gunnar Martinsson, and Joel Tropp, Finding
//	      structure with randomness: probabilistic algorithms for
//	      constructing approximate matrix decompositions,
//	      arXiv:0909.4061 [math.NA; math.PR], 2009 (available at
//	      `arXiv <http://arxiv.org/abs/0909.4061>`_).
//
//go:linkname PcaLowrank py.pca_lowrank
func PcaLowrank(A *py.Object, q *py.Object, center *py.Object, niter *py.Object) *py.Object

// Return the singular value decomposition “(U, S, V)“ of a matrix,
//
//	batches of matrices, or a sparse matrix :math:`A` such that
//	:math:`A \approx U diag(S) V^T`. In case :math:`M` is given, then
//	SVD is computed for the matrix :math:`A - M`.
//
//	.. note:: The implementation is based on the Algorithm 5.1 from
//	          Halko et al, 2009.
//
//	.. note:: To obtain repeatable results, reset the seed for the
//	          pseudorandom number generator
//
//	.. note:: The input is assumed to be a low-rank matrix.
//
//	.. note:: In general, use the full-rank SVD implementation
//	          :func:`torch.linalg.svd` for dense matrices due to its 10-fold
//	          higher performance characteristics. The low-rank SVD
//	          will be useful for huge sparse matrices that
//	          :func:`torch.linalg.svd` cannot handle.
//
//	Args::
//	    A (Tensor): the input tensor of size :math:`(*, m, n)`
//
//	    q (int, optional): a slightly overestimated rank of A.
//
//	    niter (int, optional): the number of subspace iterations to
//	                           conduct; niter must be a nonnegative
//	                           integer, and defaults to 2
//
//	    M (Tensor, optional): the input tensor's mean of size
//	                          :math:`(*, 1, n)`.
//
//	References::
//	    - Nathan Halko, Per-Gunnar Martinsson, and Joel Tropp, Finding
//	      structure with randomness: probabilistic algorithms for
//	      constructing approximate matrix decompositions,
//	      arXiv:0909.4061 [math.NA; math.PR], 2009 (available at
//	      `arXiv <https://arxiv.org/abs/0909.4061>`_).
//
//go:linkname SvdLowrank py.svd_lowrank
func SvdLowrank(A *py.Object, q *py.Object, niter *py.Object, M *py.Object) *py.Object

// unique(input, sorted=True, return_inverse=False, return_counts=False, dim=None) -> Tuple[Tensor, Tensor, Tensor]
//
//	Returns the unique elements of the input tensor.
//
//	.. note:: This function is different from :func:`torch.unique_consecutive` in the sense that
//	    this function also eliminates non-consecutive duplicate values.
//
//	.. note:: Currently in the CUDA implementation and the CPU implementation,
//	    `torch.unique` always sort the tensor at the beginning regardless of the `sort` argument.
//	    Sorting could be slow, so if your input tensor is already sorted, it is recommended to use
//	    :func:`torch.unique_consecutive` which avoids the sorting.
//
//	Args:
//	    input (Tensor): the input tensor
//	    sorted (bool): Whether to sort the unique elements in ascending order
//	        before returning as output.
//	    return_inverse (bool): Whether to also return the indices for where
//	        elements in the original input ended up in the returned unique list.
//	    return_counts (bool): Whether to also return the counts for each unique
//	        element.
//	    dim (int, optional): the dimension to operate upon. If ``None``, the
//	        unique of the flattened input is returned. Otherwise, each of the
//	        tensors indexed by the given dimension is treated as one of the
//	        elements to apply the unique operation upon. See examples for more
//	        details. Default: ``None``
//
//	Returns:
//	    (Tensor, Tensor (optional), Tensor (optional)): A tensor or a tuple of tensors containing
//
//	        - **output** (*Tensor*): the output list of unique scalar elements.
//	        - **inverse_indices** (*Tensor*): (optional) if
//	          :attr:`return_inverse` is True, there will be an additional
//	          returned tensor (same shape as input) representing the indices
//	          for where elements in the original input map to in the output;
//	          otherwise, this function will only return a single tensor.
//	        - **counts** (*Tensor*): (optional) if
//	          :attr:`return_counts` is True, there will be an additional
//	          returned tensor (same shape as output or output.size(dim),
//	          if dim was specified) representing the number of occurrences
//	          for each unique value or tensor.
//
//	Example::
//
//	    >>> output = torch.unique(torch.tensor([1, 3, 2, 3], dtype=torch.long))
//	    >>> output
//	    tensor([1, 2, 3])
//
//	    >>> output, inverse_indices = torch.unique(
//	    ...     torch.tensor([1, 3, 2, 3], dtype=torch.long), sorted=True, return_inverse=True)
//	    >>> output
//	    tensor([1, 2, 3])
//	    >>> inverse_indices
//	    tensor([0, 2, 1, 2])
//
//	    >>> output, inverse_indices = torch.unique(
//	    ...     torch.tensor([[1, 3], [2, 3]], dtype=torch.long), sorted=True, return_inverse=True)
//	    >>> output
//	    tensor([1, 2, 3])
//	    >>> inverse_indices
//	    tensor([[0, 2],
//	            [1, 2]])
//
//	    >>> a = torch.tensor([
//	    ...     [
//	    ...         [1, 1, 0, 0],
//	    ...         [1, 1, 0, 0],
//	    ...         [0, 0, 1, 1],
//	    ...     ],
//	    ...     [
//	    ...         [0, 0, 1, 1],
//	    ...         [0, 0, 1, 1],
//	    ...         [1, 1, 1, 1],
//	    ...     ],
//	    ...     [
//	    ...         [1, 1, 0, 0],
//	    ...         [1, 1, 0, 0],
//	    ...         [0, 0, 1, 1],
//	    ...     ],
//	    ... ])
//
//	    >>> # If we call `torch.unique(a, dim=0)`, each of the tensors `a[idx, :, :]`
//	    >>> # will be compared. We can see that `a[0, :, :]` and `a[2, :, :]` match
//	    >>> # each other, so one of them will be removed.
//	    >>> (a[0, :, :] == a[2, :, :]).all()
//	    tensor(True)
//	    >>> a_unique_dim0 = torch.unique(a, dim=0)
//	    >>> a_unique_dim0
//	    tensor([[[0, 0, 1, 1],
//	             [0, 0, 1, 1],
//	             [1, 1, 1, 1]],
//	            [[1, 1, 0, 0],
//	             [1, 1, 0, 0],
//	             [0, 0, 1, 1]]])
//
//	    >>> # Notice which sub-tensors from `a` match with the sub-tensors from
//	    >>> # `a_unique_dim0`:
//	    >>> (a_unique_dim0[0, :, :] == a[1, :, :]).all()
//	    tensor(True)
//	    >>> (a_unique_dim0[1, :, :] == a[0, :, :]).all()
//	    tensor(True)
//
//	    >>> # For `torch.unique(a, dim=1)`, each of the tensors `a[:, idx, :]` are
//	    >>> # compared. `a[:, 0, :]` and `a[:, 1, :]` match each other, so one of
//	    >>> # them will be removed.
//	    >>> (a[:, 0, :] == a[:, 1, :]).all()
//	    tensor(True)
//	    >>> torch.unique(a, dim=1)
//	    tensor([[[0, 0, 1, 1],
//	             [1, 1, 0, 0]],
//	            [[1, 1, 1, 1],
//	             [0, 0, 1, 1]],
//	            [[0, 0, 1, 1],
//	             [1, 1, 0, 0]]])
//
//	    >>> # For `torch.unique(a, dim=2)`, the tensors `a[:, :, idx]` are compared.
//	    >>> # `a[:, :, 0]` and `a[:, :, 1]` match each other. Also, `a[:, :, 2]` and
//	    >>> # `a[:, :, 3]` match each other as well. So in this case, two of the
//	    >>> # sub-tensors will be removed.
//	    >>> (a[:, :, 0] == a[:, :, 1]).all()
//	    tensor(True)
//	    >>> (a[:, :, 2] == a[:, :, 3]).all()
//	    tensor(True)
//	    >>> torch.unique(a, dim=2)
//	    tensor([[[0, 1],
//	             [0, 1],
//	             [1, 0]],
//	            [[1, 0],
//	             [1, 0],
//	             [1, 1]],
//	            [[0, 1],
//	             [0, 1],
//	             [1, 0]]])
//
//go:linkname Unique py.unique
func Unique(__llgo_va_list ...interface{}) *py.Object

// Converts a tensor of flat indices into a tuple of coordinate tensors that
//
//	index into an arbitrary tensor of the specified shape.
//
//	Args:
//	    indices (Tensor): An integer tensor containing indices into the
//	        flattened version of an arbitrary tensor of shape :attr:`shape`.
//	        All elements must be in the range ``[0, prod(shape) - 1]``.
//
//	    shape (int, sequence of ints, or torch.Size): The shape of the arbitrary
//	        tensor. All elements must be non-negative.
//
//	Returns:
//	    tuple of Tensors: Each ``i``-th tensor in the ouput corresponds with
//	    dimension ``i`` of :attr:`shape`. Each tensor has the same shape as
//	    ``indices`` and contains one index into dimension ``i`` for each of the
//	    flat indices given by ``indices``.
//
//	Example::
//
//	    >>> import torch
//	    >>> torch.unravel_index(torch.tensor(4), (3, 2))
//	    (tensor(2),
//	     tensor(0))
//
//	    >>> torch.unravel_index(torch.tensor([4, 1]), (3, 2))
//	    (tensor([2, 0]),
//	     tensor([0, 1]))
//
//	    >>> torch.unravel_index(torch.tensor([0, 1, 2, 3, 4, 5]), (3, 2))
//	    (tensor([0, 0, 1, 1, 2, 2]),
//	     tensor([0, 1, 0, 1, 0, 1]))
//
//	    >>> torch.unravel_index(torch.tensor([1234, 5678]), (10, 10, 10, 10))
//	    (tensor([1, 5]),
//	     tensor([2, 6]),
//	     tensor([3, 7]),
//	     tensor([4, 8]))
//
//	    >>> torch.unravel_index(torch.tensor([[1234], [5678]]), (10, 10, 10, 10))
//	    (tensor([[1], [5]]),
//	     tensor([[2], [6]]),
//	     tensor([[3], [7]]),
//	     tensor([[4], [8]]))
//
//	    >>> torch.unravel_index(torch.tensor([[1234], [5678]]), (100, 100))
//	    (tensor([[12], [56]]),
//	     tensor([[34], [78]]))
//
//go:linkname UnravelIndex py.unravel_index
func UnravelIndex(indices *py.Object, shape *py.Object) *py.Object

// Returns whether PyTorch was built with _GLIBCXX_USE_CXX11_ABI=1
//
//go:linkname CompiledWithCxx11Abi py.compiled_with_cxx11_abi
func CompiledWithCxx11Abi() *py.Object

// Find the k largest (or smallest) eigenvalues and the corresponding
//
//	eigenvectors of a symmetric positive definite generalized
//	eigenvalue problem using matrix-free LOBPCG methods.
//
//	This function is a front-end to the following LOBPCG algorithms
//	selectable via `method` argument:
//
//	  `method="basic"` - the LOBPCG method introduced by Andrew
//	  Knyazev, see [Knyazev2001]. A less robust method, may fail when
//	  Cholesky is applied to singular input.
//
//	  `method="ortho"` - the LOBPCG method with orthogonal basis
//	  selection [StathopoulosEtal2002]. A robust method.
//
//	Supported inputs are dense, sparse, and batches of dense matrices.
//
//	.. note:: In general, the basic method spends least time per
//	  iteration. However, the robust methods converge much faster and
//	  are more stable. So, the usage of the basic method is generally
//	  not recommended but there exist cases where the usage of the
//	  basic method may be preferred.
//
//	.. warning:: The backward method does not support sparse and complex inputs.
//	  It works only when `B` is not provided (i.e. `B == None`).
//	  We are actively working on extensions, and the details of
//	  the algorithms are going to be published promptly.
//
//	.. warning:: While it is assumed that `A` is symmetric, `A.grad` is not.
//	  To make sure that `A.grad` is symmetric, so that `A - t * A.grad` is symmetric
//	  in first-order optimization routines, prior to running `lobpcg`
//	  we do the following symmetrization map: `A -> (A + A.t()) / 2`.
//	  The map is performed only when the `A` requires gradients.
//
//	Args:
//
//	  A (Tensor): the input tensor of size :math:`(*, m, m)`
//
//	  B (Tensor, optional): the input tensor of size :math:`(*, m,
//	              m)`. When not specified, `B` is interpreted as
//	              identity matrix.
//
//	  X (tensor, optional): the input tensor of size :math:`(*, m, n)`
//	              where `k <= n <= m`. When specified, it is used as
//	              initial approximation of eigenvectors. X must be a
//	              dense tensor.
//
//	  iK (tensor, optional): the input tensor of size :math:`(*, m,
//	              m)`. When specified, it will be used as preconditioner.
//
//	  k (integer, optional): the number of requested
//	              eigenpairs. Default is the number of :math:`X`
//	              columns (when specified) or `1`.
//
//	  n (integer, optional): if :math:`X` is not specified then `n`
//	              specifies the size of the generated random
//	              approximation of eigenvectors. Default value for `n`
//	              is `k`. If :math:`X` is specified, the value of `n`
//	              (when specified) must be the number of :math:`X`
//	              columns.
//
//	  tol (float, optional): residual tolerance for stopping
//	             criterion. Default is `feps ** 0.5` where `feps` is
//	             smallest non-zero floating-point number of the given
//	             input tensor `A` data type.
//
//	  largest (bool, optional): when True, solve the eigenproblem for
//	             the largest eigenvalues. Otherwise, solve the
//	             eigenproblem for smallest eigenvalues. Default is
//	             `True`.
//
//	  method (str, optional): select LOBPCG method. See the
//	             description of the function above. Default is
//	             "ortho".
//
//	  niter (int, optional): maximum number of iterations. When
//	             reached, the iteration process is hard-stopped and
//	             the current approximation of eigenpairs is returned.
//	             For infinite iteration but until convergence criteria
//	             is met, use `-1`.
//
//	  tracker (callable, optional) : a function for tracing the
//	             iteration process. When specified, it is called at
//	             each iteration step with LOBPCG instance as an
//	             argument. The LOBPCG instance holds the full state of
//	             the iteration process in the following attributes:
//
//	               `iparams`, `fparams`, `bparams` - dictionaries of
//	               integer, float, and boolean valued input
//	               parameters, respectively
//
//	               `ivars`, `fvars`, `bvars`, `tvars` - dictionaries
//	               of integer, float, boolean, and Tensor valued
//	               iteration variables, respectively.
//
//	               `A`, `B`, `iK` - input Tensor arguments.
//
//	               `E`, `X`, `S`, `R` - iteration Tensor variables.
//
//	             For instance:
//
//	               `ivars["istep"]` - the current iteration step
//	               `X` - the current approximation of eigenvectors
//	               `E` - the current approximation of eigenvalues
//	               `R` - the current residual
//	               `ivars["converged_count"]` - the current number of converged eigenpairs
//	               `tvars["rerr"]` - the current state of convergence criteria
//
//	             Note that when `tracker` stores Tensor objects from
//	             the LOBPCG instance, it must make copies of these.
//
//	             If `tracker` sets `bvars["force_stop"] = True`, the
//	             iteration process will be hard-stopped.
//
//	  ortho_iparams, ortho_fparams, ortho_bparams (dict, optional):
//	             various parameters to LOBPCG algorithm when using
//	             `method="ortho"`.
//
//	Returns:
//
//	  E (Tensor): tensor of eigenvalues of size :math:`(*, k)`
//
//	  X (Tensor): tensor of eigenvectors of size :math:`(*, m, k)`
//
//	References:
//
//	  [Knyazev2001] Andrew V. Knyazev. (2001) Toward the Optimal
//	  Preconditioned Eigensolver: Locally Optimal Block Preconditioned
//	  Conjugate Gradient Method. SIAM J. Sci. Comput., 23(2),
//	  517-541. (25 pages)
//	  https://epubs.siam.org/doi/abs/10.1137/S1064827500366124
//
//	  [StathopoulosEtal2002] Andreas Stathopoulos and Kesheng
//	  Wu. (2002) A Block Orthogonalization Procedure with Constant
//	  Synchronization Requirements. SIAM J. Sci. Comput., 23(6),
//	  2165-2182. (18 pages)
//	  https://epubs.siam.org/doi/10.1137/S1064827500370883
//
//	  [DuerschEtal2018] Jed A. Duersch, Meiyue Shao, Chao Yang, Ming
//	  Gu. (2018) A Robust and Efficient Implementation of LOBPCG.
//	  SIAM J. Sci. Comput., 40(5), C655-C676. (22 pages)
//	  https://epubs.siam.org/doi/abs/10.1137/17M1129830
//
//go:linkname Lobpcg py.lobpcg
func Lobpcg(A *py.Object, k *py.Object, B *py.Object, X *py.Object, n *py.Object, iK *py.Object, niter *py.Object, tol *py.Object, largest *py.Object, method *py.Object, tracker *py.Object, orthoIparams *py.Object, orthoFparams *py.Object, orthoBparams *py.Object) *py.Object

// from_dlpack(ext_tensor) -> Tensor
//
//	Converts a tensor from an external library into a ``torch.Tensor``.
//
//	The returned PyTorch tensor will share the memory with the input tensor
//	(which may have come from another library). Note that in-place operations
//	will therefore also affect the data of the input tensor. This may lead to
//	unexpected issues (e.g., other libraries may have read-only flags or
//	immutable data structures), so the user should only do this if they know
//	for sure that this is fine.
//
//	Args:
//	    ext_tensor (object with ``__dlpack__`` attribute, or a DLPack capsule):
//	        The tensor or DLPack capsule to convert.
//
//	        If ``ext_tensor`` is a tensor (or ndarray) object, it must support
//	        the ``__dlpack__`` protocol (i.e., have a ``ext_tensor.__dlpack__``
//	        method). Otherwise ``ext_tensor`` may be a DLPack capsule, which is
//	        an opaque ``PyCapsule`` instance, typically produced by a
//	        ``to_dlpack`` function or method.
//
//	Examples::
//
//	    >>> import torch.utils.dlpack
//	    >>> t = torch.arange(4)
//
//	    # Convert a tensor directly (supported in PyTorch >= 1.10)
//	    >>> t2 = torch.from_dlpack(t)
//	    >>> t2[:2] = -1  # show that memory is shared
//	    >>> t2
//	    tensor([-1, -1,  2,  3])
//	    >>> t
//	    tensor([-1, -1,  2,  3])
//
//	    # The old-style DLPack usage, with an intermediate capsule object
//	    >>> capsule = torch.utils.dlpack.to_dlpack(t)
//	    >>> capsule
//	    <capsule object "dltensor" at ...>
//	    >>> t3 = torch.from_dlpack(capsule)
//	    >>> t3
//	    tensor([-1, -1,  2,  3])
//	    >>> t3[0] = -9  # now we're sharing memory between 3 tensors
//	    >>> t3
//	    tensor([-9, -1,  2,  3])
//	    >>> t2
//	    tensor([-9, -1,  2,  3])
//	    >>> t
//	    tensor([-9, -1,  2,  3])
//
//go:linkname FromDlpack py.from_dlpack
func FromDlpack(extTensor *py.Object) *py.Object

// Conditionally applies `true_fn` or `false_fn`.
//
// .. warning::
//
//	`torch.cond` is a prototype feature in PyTorch. It has limited support for input and output types and
//	doesn't support training currently. Please look forward to a more stable implementation in a future version of PyTorch.
//	Read more about feature classification at: https://pytorch.org/blog/pytorch-feature-classification-changes/#prototype
//
// `cond` is structured control flow operator. That is, it is like a Python if-statement,
// but has restrictions on `true_fn`, `false_fn`, and `operands` that enable it to be
// capturable using torch.compile and torch.export.
//
// Assuming the constraints on `cond`'s arguments are met, `cond` is equivalent to the following::
//
//	def cond(pred, true_branch, false_branch, operands):
//	    if pred:
//	        return true_branch(*operands)
//	    else:
//	        return false_branch(*operands)
//
// Args:
//
//	pred (Union[bool, torch.Tensor]): A boolean expression or a tensor with one element,
//	  indicating which branch function to apply.
//
//	true_fn (Callable): A callable function (a -> b) that is within the
//	  scope that is being traced.
//
//	false_fn (Callable): A callable function (a -> b) that is within the
//	  scope that is being traced. The true branch and false branch must
//	  have consistent input and outputs, meaning the inputs have to be
//	  the same, and the outputs have to be the same type and shape.
//
//	operands (Tuple of possibly nested dict/list/tuple of torch.Tensor): A tuple of inputs to the true/false functions.
//
// Example::
//
//	def true_fn(x: torch.Tensor):
//	    return x.cos()
//	def false_fn(x: torch.Tensor):
//	    return x.sin()
//	return cond(x.shape[0] > 4, true_fn, false_fn, (x,))
//
// Restrictions:
//
//   - The conditional statement (aka `pred`) must meet one of the following constraints:
//
//   - It's a `torch.Tensor` with only one element, and torch.bool dtype
//
//   - It's a boolean expression, e.g. `x.shape[0] > 10` or `x.dim() > 1 and x.shape[1] > 10`
//
//   - The branch function (aka `true_fn`/`false_fn`) must meet all of the following constraints:
//
//   - The function signature must match with operands.
//
//   - The function must return a tensor with the same metadata, e.g. shape,
//     dtype, etc.
//
//   - The function cannot have in-place mutations on inputs or global variables.
//     (Note: in-place tensor operations such as `add_` for intermediate results
//     are allowed in a branch)
//
// .. warning::
//
//	Temporal Limitations:
//
//	- `cond` only supports **inference** right now. Autograd will be supported in the future.
//
//	- The **output** of branches must be a **single Tensor**. Pytree of tensors will be supported in the future.
//
//go:linkname Cond py.cond
func Cond(pred *py.Object, trueFn *py.Object, falseFn *py.Object, operands *py.Object) *py.Object
