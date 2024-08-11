# typed: strict
# frozen_string_literal: true

module Nonoto
  module UI
    class Icon < Nonoto::UI::Base
      sig { params(name: String, color: String, size: String, class_name: String).void }
      def initialize(name:, color: "base-content", size: "16px", class_name: "")
        @name = name
        @color = color
        @size = size
        @class_name = class_name
      end

      sig { returns(String) }
      attr_reader :name
      private :name

      sig { returns(String) }
      attr_reader :color
      private :color

      sig { returns(String) }
      attr_reader :size
      private :size

      sig { returns(String) }
      attr_reader :class_name
      private :class_name

      sig { returns(String) }
      private def color_class_name
        case color
        when "base-content"
          "[&_.content]:fill-base-content"
        when "error"
          "[&_.content]:fill-error"
        when "gray-300"
          "[&_.content]:fill-gray-300"
        when "gray-400"
          "[&_.content]:fill-gray-400"
        when "gray-500"
          "[&_.content]:fill-gray-500"
        when "gray-600"
          "[&_.content]:fill-gray-600"
        when "gray-700"
          "[&_.content]:fill-gray-700"
        when "info"
          "[&_.content]:fill-info"
        when "success"
          "[&_.content]:fill-success"
        else
          fail "Unknown color: #{color.inspect}"
        end
      end

      sig { returns(String) }
      private def icon_class_name
        [class_name, color_class_name, "inline-block"].compact_blank.join(" ")
      end
    end
  end
end
