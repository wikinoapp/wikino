# typed: strict
# frozen_string_literal: true

module Views
  module Components
    module Basic
      class Dropdown
        class Menu < VC::Base
          renders_many :items, VC::Basic::Dropdown::MenuItem

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
end
