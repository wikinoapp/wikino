# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module PageSettable
    include ActionController::StrongParameters
  end
end
