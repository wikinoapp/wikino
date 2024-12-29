# typed: strict
# frozen_string_literal: true

module Views
  module Components
    module Layouts
      class Main < Views::Components::Base
        use_helpers :signed_in?
      end
    end
  end
end
