# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module SpaceSettable
    include ActionController::StrongParameters

    def request
    end
  end
end
