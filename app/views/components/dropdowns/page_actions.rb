# typed: strict
# frozen_string_literal: true

module Views
  module Components
    module Dropdowns
      class PageActions < VC::Base
        sig { params(page: Page).void }
        def initialize(page:)
          @page = page
        end

        sig { returns(Page) }
        attr_reader :page
        private :page
      end
    end
  end
end
