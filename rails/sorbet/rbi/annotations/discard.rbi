# typed: true

# DO NOT EDIT MANUALLY
# This file was pulled from a central RBI files repository.
# Please run `bin/tapioca annotations` to update it.

module Discard::Model::ClassMethods
  # @shim: defined dynamically via `define_model_callbacks :discard`
  sig { params(args: T.untyped, options: T.untyped, block: T.nilable(T.proc.bind(T.untyped).void)).void }
  def before_discard(*args, **options, &block); end

  # @shim: defined dynamically via `define_model_callbacks :discard`
  sig { params(args: T.untyped, options: T.untyped, block: T.nilable(T.proc.bind(T.untyped).void)).void }
  def around_discard(*args, **options, &block); end

  # @shim: defined dynamically via `define_model_callbacks :discard`
  sig { params(args: T.untyped, options: T.untyped, block: T.nilable(T.proc.bind(T.untyped).void)).void }
  def after_discard(*args, **options, &block); end

  # @shim: defined dynamically via `define_model_callbacks :undiscard`
  sig { params(args: T.untyped, options: T.untyped, block: T.nilable(T.proc.bind(T.untyped).void)).void }
  def before_undiscard(*args, **options, &block); end

  # @shim: defined dynamically via `define_model_callbacks :undiscard`
  sig { params(args: T.untyped, options: T.untyped, block: T.nilable(T.proc.bind(T.untyped).void)).void }
  def around_undiscard(*args, **options, &block); end

  # @shim: defined dynamically via `define_model_callbacks :undiscard`
  sig { params(args: T.untyped, options: T.untyped, block: T.nilable(T.proc.bind(T.untyped).void)).void }
  def after_undiscard(*args, **options, &block); end
end
