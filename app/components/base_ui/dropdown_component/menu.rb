# typed: strict
# frozen_string_literal: true

module BaseUI
  class DropdownComponent
    class Menu < ApplicationComponent
      renders_many :items, BaseUI::DropdownComponent::MenuItem

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
        class_names(
          "bg-base-300 dropdown-content menu rounded-box shadow z-1",
          options[:class]
        )
      end

      sig { returns(T::Hash[Symbol, String]) }
      private def build_options
        options.merge(
          class: build_class_name,
          role: "menu",
          "aria-labelledby": button_id
        )
      end
    end
  end
end
