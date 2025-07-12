# typed: strict
# frozen_string_literal: true

module BaseUI
  class DropdownComponent
    class MenuItem < ApplicationComponent
      sig { params(options: T::Hash[Symbol, String]).void }
      def initialize(options: {})
        @options = options
      end

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
          role: "menuitem"
        )
      end
    end
  end
end
