# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module SpaceAware
    include ControllerConcerns::Authenticatable
    include ActionController::StrongParameters

    sig { returns(T.untyped) }
    def params
    end
  end
end
