# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module Authorizable
    include ActionController::StrongParameters
    include Pundit::Authorization
  end
end
