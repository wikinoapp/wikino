# typed: strict
# frozen_string_literal: true

module Cards
  class EmptyStateCardComponent < ApplicationComponent
    sig { params(icon_name: String, message: String, class_name: String).void }
    def initialize(icon_name:, message:, class_name: "")
      @icon_name = icon_name
      @message = message
      @class_name = class_name
    end

    sig { returns(String) }
    attr_reader :icon_name
    private :icon_name

    sig { returns(String) }
    attr_reader :message
    private :message

    sig { returns(String) }
    attr_reader :class_name
    private :class_name
  end
end
