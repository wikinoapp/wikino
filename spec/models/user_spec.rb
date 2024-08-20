# typed: false
# frozen_string_literal: true

RSpec.describe User, type: :model do
  describe "#viewable_lists" do
    context do
      let!(:space) { create(:space) }
      let!(:viewer) { create(:user, space:) }
      let!(:list_a) { create(:list, :public, space:, name: "リストA") }
      let!(:list_b) { create(:list, :private, space:, name: "リストB") }
      let!(:list_c) { create(:list, :private, space:, name: "リストC") }
      let!(:list_d) { create(:list, :private, space:, name: "リストD") }
      let!(:list_c_member_a) { create(:list_member, :admin, space:, list: list_c, user: viewer) }
      let!(:list_d_member_a) { create(:list_member, :member, space:, list: list_d, user: viewer) }

      it do
        expect(viewer.viewable_lists).to contain_exactly(list_a, list_c, list_d)
      end
    end
  end

  describe "#last_note_modified_lists" do
    context do
      let!(:space) { create(:space) }
      let!(:viewer) { create(:user, space:) }
      let!(:list_a) { create(:list, space:, name: "リストA") }
      let!(:list_b) { create(:list, space:, name: "リストB") }
      let!(:list_c) { create(:list, space:, name: "リストC") }
      let!(:list_d) { create(:list, space:, name: "リストD") }
      let!(:list_e) { create(:list, space:, name: "リストE") }
      let!(:list_a_member_a) { create(:list_member, space:, list: list_a, user: viewer, joined_at: Time.parse("2024-08-18 0:00:00"), last_note_modified_at: nil) }
      let!(:list_b_member_a) { create(:list_member, space:, list: list_b, user: viewer, joined_at: Time.parse("2024-08-18 1:00:00"), last_note_modified_at: Time.parse("2024-08-19 0:00:00")) }
      let!(:list_c_member_a) { create(:list_member, space:, list: list_c, user: viewer, joined_at: Time.parse("2024-08-18 2:00:00"), last_note_modified_at: Time.parse("2024-08-19 1:00:00")) }
      let!(:list_d_member_a) { create(:list_member, space:, list: list_d, user: viewer, joined_at: Time.parse("2024-08-18 3:00:00"), last_note_modified_at: nil) }

      it do
        expect(
          viewer.last_note_modified_lists.pluck(:name)
        ).to eq(["リストC", "リストB", "リストD", "リストA"])
      end
    end
  end

  describe "#last_modified_notes" do
    context do
      let!(:space) { create(:space) }
      let!(:user_a) { create(:user, space:) }
      let!(:user_b) { create(:user, space:) }
      let!(:list) { create(:list, space:) }
      let!(:note_a) { create(:note, space:, list:, author: user_a, title: "ノートA") }
      let!(:note_b) { create(:note, space:, list:, author: user_b, title: "ノートB") }
      let!(:note_c) { create(:note, space:, list:, author: user_b, title: "ノートC") }
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
