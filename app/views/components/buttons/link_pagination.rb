# typed: strict
# frozen_string_literal: true

module Views
  module Components
    module Buttons
      class LinkPagination < VC::Base
        sig { params(path: String).void }
        def initialize(path:)
          @path = path
        end

        sig { returns(String) }
        attr_reader :path
        private :path
      end
    end
  end
end
