# typed: strict
# frozen_string_literal: true

module BaseUI
  class LabelComponent < ApplicationComponent
    sig do
      params(
        form_builder: ActionView::Helpers::FormBuilder,
        method: Symbol,
        optional: T::Boolean
      ).void
    end
    def initialize(form_builder:, method:, optional: false)
      @form_builder = form_builder
      @method = method
      @optional = optional
    end

    sig { returns(ActionView::Helpers::FormBuilder) }
    attr_reader :form_builder
    private :form_builder

    sig { returns(Symbol) }
    attr_reader :method
    private :method

    sig { returns(T::Boolean) }
    attr_reader :optional
    private :optional
    alias_method :optional?, :optional
  end
end
