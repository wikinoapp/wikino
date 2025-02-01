# typed: strict
# frozen_string_literal: true

module ModelConcerns
  module SpaceViewable
    extend T::Sig
    extend T::Helpers

    interface!

    sig { abstract.returns(Page::PrivateRelation) }
    def viewable_pages
    end

    sig { abstract.returns(Topic::PrivateRelation) }
    def topics
    end

    sig { abstract.returns(T::Boolean) }
    def can_create_topic?
    end
  end
end
