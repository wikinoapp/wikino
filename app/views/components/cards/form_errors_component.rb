# typed: strict
# frozen_string_literal: true

module Cards
  class FormErrorsComponent < ApplicationComponent
    sig { params(errors: ActiveModel::Errors, class_name: String).void }
    def initialize(errors:, class_name: "")
      @errors = errors
      @class_name = class_name
    end

    sig { returns(String) }
    attr_reader :class_name
    private :class_name

    sig { returns(T::Boolean) }
    private def render?
      !@errors.empty?
    end
  end
end
