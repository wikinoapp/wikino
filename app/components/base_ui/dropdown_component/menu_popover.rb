# typed: strict
# frozen_string_literal: true

module BaseUI
  class DropdownComponent
    class MenuPopover < ApplicationComponent
      renders_one :menu, ->(options: {}) do
        BaseUI::DropdownComponent::Menu.new(button_id:, options:)
      end

      sig do
        params(
          button_id: String,
          popover_id: String,
          menu_align: BaseUI::DropdownComponent::MenuAlign,
          options: T::Hash[Symbol, String]
        ).void
      end
      def initialize(button_id:, popover_id:, menu_align:, options: {})
        @button_id = button_id
        @popover_id = popover_id
        @menu_align = menu_align
        @options = options
      end

      sig { returns(String) }
      attr_reader :button_id
      private :button_id

      sig { returns(String) }
      attr_reader :popover_id
      private :popover_id

      sig { returns(BaseUI::DropdownComponent::MenuAlign) }
      attr_reader :menu_align
      private :menu_align

      sig { returns(T::Hash[Symbol, String]) }
      attr_reader :options
      private :options

      sig { returns(String) }
      private def build_class_name
        options[:class]
      end

      sig { returns(T::Hash[Symbol, String]) }
      private def build_options
        options.merge(
          class: build_class_name,
          id: popover_id,
          "aria-hidden": "true",
          "data-align": menu_align.serialize,
          "data-popover": "true"
        )
      end
    end
  end
end
