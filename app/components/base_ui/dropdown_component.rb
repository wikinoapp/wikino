# typed: strict
# frozen_string_literal: true

module BaseUI
  class DropdownComponent < ApplicationComponent
    class MenuAlign < T::Enum
      enums do
        Start = new
        End = new
      end
    end

    renders_one :button, ->(options: {}) do
      BaseUI::DropdownComponent::Button.new(button_id:, options:)
    end

    renders_one :menu_popover, ->(options: {}) do
      BaseUI::DropdownComponent::MenuPopover.new(button_id:, popover_id:, menu_align:, options:)
    end

    sig { params(id: String, menu_align: MenuAlign, options: T::Hash[Symbol, String]).void }
    def initialize(id:, menu_align: MenuAlign::End, options: {})
      @id = id
      @menu_align = menu_align
      @options = options
    end

    sig { returns(String) }
    attr_reader :id
    private :id

    sig { returns(MenuAlign) }
    attr_reader :menu_align
    private :menu_align

    sig { returns(T::Hash[Symbol, String]) }
    attr_reader :options
    private :options

    sig { returns(String) }
    private def build_class_name
      class_names("dropdown-menu", options[:class])
    end

    sig { returns(T::Hash[Symbol, String]) }
    private def build_options
      options.merge(
        class: build_class_name,
        id:
      )
    end

    sig { returns(String) }
    private def button_id
      "#{id}-trigger"
    end

    sig { returns(String) }
    private def popover_id
      "#{id}-popover"
    end
  end
end
