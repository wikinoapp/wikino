# typed: strict
# frozen_string_literal: true

module Views
  module Components
    module Sidebars
      class Main < Views::Components::Base
        use_helpers :signed_in?

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
