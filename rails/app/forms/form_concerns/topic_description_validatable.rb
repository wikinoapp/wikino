# typed: strict
# frozen_string_literal: true

module FormConcerns
  module TopicDescriptionValidatable
    extend T::Sig
    extend ActiveSupport::Concern

    included do
      include ActiveModel::Validations::Callbacks

      before_validation :convert_nil_to_empty_string
      validates :description, length: {maximum: Topic::DESCRIPTION_MAX_LENGTH}
    end

    sig { void }
    private def convert_nil_to_empty_string
      self.description = "" if description.nil?
    end
  end
end
