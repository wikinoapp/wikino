# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module SpaceSettable
    include ActionController::StrongParameters
  end
end
