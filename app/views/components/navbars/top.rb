# typed: strict
# frozen_string_literal: true

module Views
  module Components
    module Navbars
      class Top < VC::Base
        use_helpers :signed_in?
      end
    end
  end
end
