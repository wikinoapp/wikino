# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module TopicSettable
    include ActionController::StrongParameters

    def authorize(*args)
    end
  end
end
