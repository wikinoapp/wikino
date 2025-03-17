# typed: strict
# frozen_string_literal: true

module FormConcerns
  module TopicVisibilityValidatable
    extend ActiveSupport::Concern

    included do
      validates :visibility, inclusion: {in: Topic.visibilities.keys}, presence: true
    end
  end
end
