# typed: strict
# frozen_string_literal: true

module BaseUI
  module Card
    class BodyComponent < ApplicationComponent
      sig { params(class_name: String).void }
      def initialize(class_name: "")
        @class_name = class_name
      end

      sig { returns(String) }
      attr_reader :class_name
      private :class_name
    end
  end
end
