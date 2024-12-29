# typed: strict
# frozen_string_literal: true

module Views
  module Components
    module Basic
      class Card < VC::Base
        renders_one :body, ->(class_name: "") do
          VC::Card::Body.new(class_name:)
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
  end
end
