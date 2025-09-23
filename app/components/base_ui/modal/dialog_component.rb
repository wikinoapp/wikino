# typed: strict
# frozen_string_literal: true

module BaseUI
  module Modal
    class DialogComponent < ApplicationComponent
      sig { params(controller_name: String, target_name: String, class_name: String).void }
      def initialize(controller_name: "dialog", target_name: "dialog", class_name: "")
        @controller_name = controller_name
        @target_name = target_name
        @class_name = class_name
      end

      sig { returns(String) }
      attr_reader :controller_name
      private :controller_name

      sig { returns(String) }
      attr_reader :class_name
      private :class_name

      sig { void }
      def dialog_class_name
        [
          "dialog",
          "w-full sm:max-w-2xl",
          "rounded-lg",
          "bg-white",
          "p-0",
          "shadow-xl",
          "backdrop:bg-black/50",
          class_name
        ].compact.join(" ")
      end
    end
  end
end
