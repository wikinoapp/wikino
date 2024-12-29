# typed: strict
# frozen_string_literal: true

module Views
  module Components
    module Toasts
      class Flash < VC::Base
        sig { params(flash: ActionDispatch::Flash::FlashHash).void }
        def initialize(flash:)
          @flash = flash
        end

        sig { returns(T.nilable(Symbol)) }
        private def flash_toast_type
          @flash.keys.first&.to_sym
        end
      end
    end
  end
end
