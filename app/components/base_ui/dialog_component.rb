# typed: strict
# frozen_string_literal: true

module BaseUI
  class DialogComponent < ApplicationComponent
    sig { params(id: String, class_name: String).void }
    def initialize(id:, class_name: "")
      @id = id
      @class_name = class_name
    end

    sig { returns(String) }
    def dom_id
      @id
    end

    sig { returns(String) }
    def dialog_classes
      [
        "dialog",
        "w-full sm:max-w-2xl",
        "rounded-lg",
        "bg-white",
        "p-0",
        "shadow-xl",
        "backdrop:bg-black/50",
        @class_name
      ].compact.join(" ")
    end

    private

    sig { returns(String) }
    attr_reader :id

    sig { returns(String) }
    attr_reader :class_name
  end
end
