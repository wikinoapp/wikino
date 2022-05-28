# typed: strict
# frozen_string_literal: true

module Forms
  class ApplicationForm
    extend T::Sig
    include ActiveModel::Model

    # @overload
    sig { returns(Symbol) }
    def self.i18n_scope
      :form
    end
  end
end
