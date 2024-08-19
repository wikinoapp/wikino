# typed: false
# frozen_string_literal: true

RSpec.describe User, type: :model do
  describe "#viewable_notebooks" do
    context do
      let!(:space) { create(:space) }
      let!(:viewer) { create(:user, space:) }
      let!(:notebook_a) { create(:notebook, :public, space:, name: "ノートブックA") }
      let!(:notebook_b) { create(:notebook, :private, space:, name: "ノートブックB") }
      let!(:notebook_c) { create(:notebook, :private, space:, name: "ノートブックC") }
      let!(:notebook_d) { create(:notebook, :private, space:, name: "ノートブックD") }
      let!(:notebook_c_member_a) { create(:notebook_member, :admin, space:, notebook: notebook_c, user: viewer) }
      let!(:notebook_d_member_a) { create(:notebook_member, :member, space:, notebook: notebook_d, user: viewer) }

      it do
        expect(viewer.viewable_notebooks).to contain_exactly(notebook_a, notebook_c, notebook_d)
      end
    end
  end

  describe "#last_note_modified_notebooks" do
    context do
      let!(:space) { create(:space) }
      let!(:viewer) { create(:user, space:) }
      let!(:notebook_a) { create(:notebook, space:, name: "ノートブックA") }
      let!(:notebook_b) { create(:notebook, space:, name: "ノートブックB") }
      let!(:notebook_c) { create(:notebook, space:, name: "ノートブックC") }
      let!(:notebook_d) { create(:notebook, space:, name: "ノートブックD") }
      let!(:notebook_e) { create(:notebook, space:, name: "ノートブックE") }
      let!(:notebook_a_member_a) { create(:notebook_member, space:, notebook: notebook_a, user: viewer, joined_at: Time.parse("2024-08-18 0:00:00"), last_note_modified_at: nil) }
      let!(:notebook_b_member_a) { create(:notebook_member, space:, notebook: notebook_b, user: viewer, joined_at: Time.parse("2024-08-18 1:00:00"), last_note_modified_at: Time.parse("2024-08-19 0:00:00")) }
      let!(:notebook_c_member_a) { create(:notebook_member, space:, notebook: notebook_c, user: viewer, joined_at: Time.parse("2024-08-18 2:00:00"), last_note_modified_at: Time.parse("2024-08-19 1:00:00")) }
      let!(:notebook_d_member_a) { create(:notebook_member, space:, notebook: notebook_d, user: viewer, joined_at: Time.parse("2024-08-18 3:00:00"), last_note_modified_at: nil) }

      it do
        expect(
          viewer.last_note_modified_notebooks.pluck(:name)
        ).to eq(["ノートブックC", "ノートブックB", "ノートブックD", "ノートブックA"])
      end
    end
  end

  describe "#last_modified_notes" do
    context do
      let!(:space) { create(:space) }
      let!(:user_a) { create(:user, space:) }
      let!(:user_b) { create(:user, space:) }
      let!(:notebook) { create(:notebook, space:) }
      let!(:note_a) { create(:note, space:, notebook:, author: user_a, title: "ノートA") }
      let!(:note_b) { create(:note, space:, notebook:, author: user_b, title: "ノートB") }
      let!(:note_c) { create(:note, space:, notebook:, author: user_b, title: "ノートC") }
      let!(:note_a_editor_a) { create(:note_editor, space:, note: note_a, user: user_a, last_note_modified_at: Time.parse("2024-08-18 0:00:00")) }
      let!(:note_b_editor_a) { create(:note_editor, space:, note: note_b, user: user_b, last_note_modified_at: Time.parse("2024-08-18 1:00:00")) }
      let!(:note_c_editor_a) { create(:note_editor, space:, note: note_c, user: user_b, last_note_modified_at: Time.parse("2024-08-18 2:00:00")) }
      let!(:note_c_editor_b) { create(:note_editor, space:, note: note_c, user: user_a, last_note_modified_at: Time.parse("2024-08-18 3:00:00")) }

      it do
        expect(user_a.last_modified_notes.pluck(:title)).to eq(["ノートC", "ノートA"])
      end
    end
  end
end
