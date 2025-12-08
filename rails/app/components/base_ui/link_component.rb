# typed: strict
# frozen_string_literal: true

module BaseUI
  class LinkComponent < ApplicationComponent
    class Variant < T::Enum
      enums do
        Plain = new
        Underline = new
      end
    end

    sig do
      params(
        variant: Variant,
        options: T::Hash[Symbol, String]
      ).void
    end
    def initialize(variant: Variant::Plain, options: {})
      @variant = variant
      @options = options
    end

    sig { returns(Variant) }
    attr_reader :variant
    private :variant

    sig { returns(T::Hash[Symbol, String]) }
    attr_reader :options
    private :options

    sig { returns(String) }
    private def build_class_name
      class_names(
        options[:class],
        "underline underline-offset-4": variant == Variant::Underline
      )
    end

    sig { returns(T::Hash[Symbol, String]) }
    private def build_options
      options.merge(
        class: build_class_name
      )
    end
  end
end
