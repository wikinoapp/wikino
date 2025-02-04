# typed: strict
# frozen_string_literal: true

module ModelConcerns
  module SpaceViewable
    extend T::Sig
    extend T::Helpers

    interface!

    sig { abstract.returns(Page::PrivateAssociationRelation) }
    def viewable_pages
    end

    sig { abstract.returns(Topic::PrivateAssociationRelation) }
    def topics
    end

    sig { abstract.returns(T::Boolean) }
    def can_create_topic?
    end

    sig { abstract.params(page: Page).returns(T::Boolean) }
    def can_update_draft_page?(page:)
    end
  end
end
