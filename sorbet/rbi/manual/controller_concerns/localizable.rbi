# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module Localizable
    include ActionController::StrongParameters

    def http_accept_language
    end

    def viewer
    end
  end
end
