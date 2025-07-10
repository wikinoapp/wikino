# typed: strict
# frozen_string_literal: true

module BaseUI
  class DropdownComponent
    class MenuPopover < ApplicationComponent
      renders_one :menu, ->(options: {}) do
        BaseUI::DropdownComponent::Menu.new(button_id:, options:)
      end

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
        options[:class]
      end

      sig { returns(T::Hash[Symbol, String]) }
      private def build_options
        options.merge(
          class: build_class_name,
          data_popover: "true",
          aria_hidden: "true"
        )
      end
    end
  end
end
