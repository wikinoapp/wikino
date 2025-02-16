# typed: strict
# frozen_string_literal: true

module FormConcerns
  module ISpace
    include Kernel

    extend T::Sig
    extend T::Helpers

    interface!

    sig { abstract.params(identifier: String).returns(T::Boolean) }
    def identifier_uniqueness?(identifier)
    end
  end
end
