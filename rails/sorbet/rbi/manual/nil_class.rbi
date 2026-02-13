# typed: strong
# frozen_string_literal: true

class NilClass
  sig { returns(T.noreturn) }
  def not_nil!
  end
end
