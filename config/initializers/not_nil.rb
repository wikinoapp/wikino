# typed: strict
# frozen_string_literal: true

module Kernel
  def not_nil!
    self
  end
end

class NilClass
  def not_nil!
    raise TypeError.new("Expected not to be nil")
  end
end
