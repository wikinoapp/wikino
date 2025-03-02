# typed: strict
# frozen_string_literal: true

module ModelConcerns
  module SpaceViewable
    extend T::Sig
    extend T::Helpers

    interface!

    sig { abstract.returns(Page::PrivateAssociationRelation) }
    def showable_pages
    end

    sig { abstract.returns(T.any(Topic::PrivateAssociationRelation, Topic::PrivateRelation)) }
    def joined_topics
    end

    sig { abstract.returns(Topic::PrivateAssociationRelation) }
    def showable_topics
    end

    sig { abstract.params(space: Space).returns(T::Boolean) }
    def can_update_space?(space:)
    end

    sig { abstract.params(topic: T.nilable(Topic)).returns(T::Boolean) }
    def can_create_page?(topic:)
    end

    sig { abstract.params(space: Space).returns(T::Boolean) }
    def can_create_bulk_restored_pages?(space:)
    end

    sig { abstract.params(page: Page).returns(T::Boolean) }
    def can_view_page?(page:)
    end

    sig { abstract.params(space: Space).returns(T::Boolean) }
    def can_view_trash?(space:)
    end

    sig { abstract.params(page: Page).returns(T::Boolean) }
    def can_update_page?(page:)
    end

    sig { abstract.returns(T::Boolean) }
    def can_create_topic?
    end

    sig { abstract.params(page: Page).returns(T::Boolean) }
    def can_update_draft_page?(page:)
    end
  end
end
