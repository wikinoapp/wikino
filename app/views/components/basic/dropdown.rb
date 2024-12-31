# typed: strict
# frozen_string_literal: true

module Views
  module Components
    module Basic
      class Dropdown < VC::Base
        renders_one :button, ->(class_name: "") do
          VC::Basic::Dropdown::Button.new(class_name:)
        end

        renders_one :menu, ->(class_name: "") do
          VC::Basic::Dropdown::Menu.new(class_name:)
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
