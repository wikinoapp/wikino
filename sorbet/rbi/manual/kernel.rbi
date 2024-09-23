# typed: strong
# frozen_string_literal: true

module Kernel
  sig { returns(T.self_type) }
  def not_nil!
  end
end
