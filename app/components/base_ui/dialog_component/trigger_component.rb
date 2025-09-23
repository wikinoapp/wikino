# typed: strict
# frozen_string_literal: true

module BaseUI
  class DialogComponent
    class TriggerComponent < ApplicationComponent
      sig { params(controller_name: String, target_name: String, action_name: String).void }
      def initialize(controller_name: "dialog", target_name: "trigger", action_name: "open")
        @controller_name = controller_name
        @target_name = target_name
        @action_name = action_name
      end

      sig { returns(String) }
      attr_reader :controller_name
      private :controller_name

      sig { returns(String) }
      attr_reader :target_name
      private :target_name

      sig { returns(String) }
      attr_reader :action_name
      private :action_name
    end
  end
end
