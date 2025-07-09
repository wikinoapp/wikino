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
        href: String,
        variant: Variant,
        class_name: String,
        tabindex: T.nilable(Integer)
      ).void
    end
    def initialize(href:, variant: Variant::Plain, class_name: "", tabindex: nil)
      @href = href
      @variant = variant
      @class_name = class_name
      @tabindex = tabindex
    end

    sig { returns(String) }
    attr_reader :href
    private :href

    sig { returns(Variant) }
    attr_reader :variant
    private :variant

    sig { returns(String) }
    attr_reader :class_name
    private :class_name

    sig { returns(T.nilable(Integer)) }
    attr_reader :tabindex
    private :tabindex

    sig { returns(String) }
    private def build_class_name
      class_names(
        class_name,
        "underline underline-offset-4": variant == Variant::Underline
      )
    end
  end
end
