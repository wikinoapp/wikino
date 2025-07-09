# typed: strict
# frozen_string_literal: true

module BaseUI
  class DropdownComponent < ApplicationComponent
    renders_one :button, ->(class_name: "") do
      BaseUI::Dropdown::ButtonComponent.new(class_name:)
    end

    renders_one :menu, ->(class_name: "") do
      BaseUI::Dropdown::MenuComponent.new(class_name:)
    end

    sig { params(class_name: String).void }
    def initialize(class_name: "")
      @class_name = class_name
    end

    sig { returns(String) }
    attr_reader :class_name
    private :class_name
  end
end
