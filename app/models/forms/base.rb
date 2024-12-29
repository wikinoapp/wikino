# typed: strict
# frozen_string_literal: true

module Forms
  class Base
    extend T::Sig

    include ActiveModel::Model
    include ActiveModel::Attributes

    # @overload
    sig { returns(Symbol) }
    def self.i18n_scope
      :forms
    end
  end
end
