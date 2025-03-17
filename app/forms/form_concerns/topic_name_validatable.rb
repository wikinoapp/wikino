# typed: strict
# frozen_string_literal: true

module FormConcerns
  module TopicNameValidatable
    extend ActiveSupport::Concern

    included do
      validates :name, length: {maximum: Topic::NAME_MAX_LENGTH}, presence: true
    end
  end
end
