# typed: strict
# frozen_string_literal: true

module FormConcerns
  module TopicVisibilityValidatable
    extend ActiveSupport::Concern

    included do
      validates :visibility,
        inclusion: {
          in: TopicVisibility.values.map(&:serialize)
        },
        presence: true
    end
  end
end
