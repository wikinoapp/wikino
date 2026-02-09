# typed: strict
# frozen_string_literal: true

module BaseUI
  class DropdownComponent
    class Button < ApplicationComponent
      sig { params(button_id: String, options: T::Hash[Symbol, String]).void }
      def initialize(button_id:, options: {})
        @button_id = button_id
        @options = options
      end

      sig { returns(String) }
      attr_reader :button_id
      private :button_id

      sig { returns(T::Hash[Symbol, String]) }
      attr_reader :options
      private :options

      sig { returns(String) }
      private def build_class_name
        options[:class].presence || ""
      end

      sig { returns(T::Hash[Symbol, String]) }
      private def build_options
        options.merge(
          class: build_class_name,
          type: "button",
          id: button_id,
          "aria-haspopup": "menu",
          "aria-controls": "demo-dropdown-menu-menu",
          "aria-expanded": "false"
        )
      end
    end
  end
end
