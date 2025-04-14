# typed: strict
# frozen_string_literal: true

module ModelConcerns
  module SpaceViewable
    extend T::Sig
    extend T::Helpers

    interface!

    sig { abstract.returns(T.any(DraftPageRecord::PrivateAssociationRelation, DraftPageRecord::PrivateRelation)) }
    def draft_pages
    end

    sig { abstract.returns(PageRecord::PrivateAssociationRelation) }
    def showable_pages
    end

    sig { abstract.returns(T.any(TopicRecord::PrivateAssociationRelation, TopicRecord::PrivateRelation)) }
    def joined_topics
    end

    sig { abstract.returns(TopicRecord::PrivateAssociationRelation) }
    def showable_topics
    end

    sig { abstract.params(space: SpaceRecord).returns(T::Boolean) }
    def can_update_space?(space:)
    end

    sig { abstract.params(space: SpaceRecord).returns(T::Boolean) }
    def can_export_space?(space:)
    end

    sig { abstract.params(topic: TopicRecord).returns(T::Boolean) }
    def can_update_topic?(topic:)
    end

    sig { abstract.params(topic: T.nilable(TopicRecord)).returns(T::Boolean) }
    def can_create_page?(topic:)
    end

    sig { abstract.params(space: SpaceRecord).returns(T::Boolean) }
    def can_create_bulk_restored_pages?(space:)
    end

    sig { abstract.params(page: PageRecord).returns(T::Boolean) }
    def can_view_page?(page:)
    end

    sig { abstract.params(space: SpaceRecord).returns(T::Boolean) }
    def can_view_trash?(space:)
    end

    sig { abstract.params(page: PageRecord).returns(T::Boolean) }
    def can_update_page?(page:)
    end

    sig { abstract.returns(T::Boolean) }
    def can_create_topic?
    end

    sig { abstract.params(page: PageRecord).returns(T::Boolean) }
    def can_update_draft_page?(page:)
    end
  end
end
