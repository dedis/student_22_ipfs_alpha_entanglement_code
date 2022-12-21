import only_data
import data_and_parity
import different_nb_nodes
import different_file_size


def main():
    save = False

    # ONLY DATA MISSING
    only_data.overhead(save)

    # DATA ALL MISSING, PART OF PARITY MISSING
    data_and_parity.download_overhead(save)
    data_and_parity.memory_overhead(save)
    data_and_parity.recovery_likelihood(save)
    data_and_parity.best_effort(save)

    # DIFFERENT FILE SIZE
    different_file_size.download_overhead(save)
    different_file_size.memory_overhead(save)
    different_file_size.recovery_likelihood(save)
    different_file_size.best_effort(save)

    # DIFFERENT NUMBER OF NODES
    different_nb_nodes.download_overhead(save)
    different_nb_nodes.memory_overhead(save)
    different_nb_nodes.recovery_likelihood(save)
    different_nb_nodes.best_effort(save)


if __name__ == "__main__":
    main()
