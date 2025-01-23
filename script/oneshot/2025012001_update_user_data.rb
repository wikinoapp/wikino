# frozen_string_literal: true

User.find_each do |user|
  space = user.space

  ActiveRecord::Base.transaction do
    space_member = user.space_members.create!(
      space:,
      role: SpaceMemberRole::Owner.serialize,
      joined_at: space.joined_at
    )

    space.draft_pages.update_all(editor_id: space_member.id)
    space.page_editorships.update_all(editor_id: space_member.id)
    space.page_revisions.update_all(editor_id: space_member.id)
    space.topic_memberships.update_all(member_id: space_member.id)
  end
end
