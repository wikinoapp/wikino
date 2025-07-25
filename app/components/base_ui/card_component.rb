# typed: strict
# frozen_string_literal: true

module BaseUI
  class CardComponent < ApplicationComponent
    renders_one :body, ->(class_name: "") do
      BaseUI::Card::BodyComponent.new(class_name:)
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
